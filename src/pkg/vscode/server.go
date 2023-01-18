package vscode

import (
	"aliyun/serverless/webide-server/pkg/context"
	"aliyun/serverless/webide-server/pkg/tar"
	"bytes"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/golang/glog"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

type (
	Server struct {
		Host              string // vscode server host
		Port              string // vscode server port
		VscodeDataDir     string // vscode server directory to store the user, server and extension data
		VscodeBinaryDir   string // the directory where vscode binary is
		WorkspaceDir      string // the workspace directory
		OssBucketName     string // oss bucket to persist the whole data
		VscodeDataOssPath string // oss path where store the vscode server data
		WorkspaceOssPath  string // oss path where store the user workspace data
		OssClient         *oss.Client
	}
	ServerOption func(*Server)
)

// NewServer creates the vscode server.
// ctx contains info, such as the ak_id/secret credential info, that is generated at runtime.
// configFilePath is the config file where store the configuration for vscode server running.
func NewServer(ctx *context.Context) (*Server, error) {
	// Read the configurations from the specified file.
	// err := viper.ReadInConfig()
	// if err != nil {
	// 	glog.Errorf("Failed to read vscode server config file. Error: %v", err)
	// 	return nil, err
	// }

	// Set the default values for each configuration item.
	viper.SetDefault("vscode.host", "127.0.0.1")
	viper.SetDefault("vscode.port", "9527")
	viper.SetDefault("vscode.dataDirectory", "~/.config/vscode-server")
	viper.SetDefault("vscode.binaryDirectory", "")
	viper.SetDefault("vscode.dataOssPath", "")
	viper.SetDefault("workspace.directory", "/workspace")
	viper.SetDefault("workspace.ossPath", "")
	viper.SetDefault("ossBucketName", "")

	s := &Server{}
	s.Host = viper.GetString("vscode.host")
	s.Port = viper.GetString("vscode.port")
	s.VscodeDataDir, _ = homedir.Expand(viper.GetString("vscode.dataDirectory"))
	s.VscodeBinaryDir, _ = homedir.Expand(viper.GetString("vscode.binaryDirectory"))
	s.VscodeDataOssPath = viper.GetString("vscode.dataOssPath")
	s.WorkspaceDir, _ = homedir.Expand(viper.GetString("workspace.directory"))
	s.WorkspaceOssPath = viper.GetString("workspace.ossPath")
	s.OssBucketName = viper.GetString("ossBucketName")

	// high priority env
	OssBucketName := os.Getenv("OSS_BUCKET_NAME")
	if OssBucketName != "" {
		s.OssBucketName = OssBucketName
	}

	glog.Infof("Read vscode server config succeeded. Server config: %+v", *s)

	ossEndpoint := "https://oss-" + ctx.Region + ".aliyuncs.com"
	c, err := oss.New(ossEndpoint, ctx.AccessKeyId, ctx.AccessKeySecret, oss.SecurityToken(ctx.SecurityToken))
	if err != nil {
		glog.Errorf("Create oss client failed. Context: %+v Error: %v", ctx, err)
		return nil, err
	}
	s.OssClient = c

	if err = s.init(); err != nil {
		glog.Errorf("Init vscode server failed. Error: %v", err)
		return nil, err
	}

	glog.Infof("Init vscode server succeeded.")

	return s, nil
}

// init the vscode server.
func (s *Server) init() error {
	// Load workspace from oss.
	workspaceLoadingResult := make(chan error)
	go func() {
		err := s.load(s.WorkspaceOssPath, s.WorkspaceDir)
		workspaceLoadingResult <- err
	}()

	// Load vscode server data from oss.
	if err := s.load(s.VscodeDataOssPath, s.VscodeDataDir); err != nil {
		glog.Errorf("Load vscode server data from oss failed. Vscode server: %+v Error: %v", *s, err)
		return err
	}
	glog.Infof("Load vscode server data from oss succeeded.")

	// Make sure vscode server is ready for recive the requests.

	// Launch the vscode server.
	// Make sure the openvscode-server binary in the system path.
	userDataDir := filepath.Join(s.VscodeDataDir, "user-data")
	serverDataDir := filepath.Join(s.VscodeDataDir, "server-data")
	extensionsDir := filepath.Join(s.VscodeDataDir, "extensions")
	// if use token auth, "--connection-token=<my token>"
	cmd := exec.Command(
		filepath.Join(s.VscodeBinaryDir, "openvscode-server"),
		"--host="+s.Host, "--port="+s.Port,
		"--user-data-dir="+userDataDir, "--server-data-dir="+serverDataDir, "--extensions-dir="+extensionsDir,
		"--without-connection-token", "--start-server", "--telemetry-level=off", "--default-folder="+s.WorkspaceDir)
	err := cmd.Start()
	if err != nil {
		glog.Errorf("Launch vscode server failed. cmd: %s error: %v", cmd.String(), err)
		return err
	}
	glog.Infof("Launch vscode server succeeded. Cmd: %s", cmd.String())

	for {
		if _, err := net.Dial("tcp", s.Host+":"+s.Port); err == nil {
			glog.Infof("Vscode server ready for recive requests.")
			break
		} else {
			glog.Infof("Waiting for vscode server ready: %v", err)
			time.Sleep(30 * time.Millisecond)
		}
	}

	// Wait for the workspace loading goroutine done.
	// Ideally, vscode server launching should not be blocked by workspace loading.
	// User should see vscode in browser very quickly and an on-going workspace loading in vscode web ide, like what did in vscode.dev for loading github project.
	// TODO: Optimize out the waiting for workspace loading.
	err = <-workspaceLoadingResult
	if err != nil {
		return err
	}

	glog.Infof("Load workspace data from oss succeeded.")

	return nil
}

// Shutdown shut down the vscode server.
func (s *Server) Shutdown() {
	// Save the vscode server data to oss.
	err := s.save(s.VscodeDataDir, s.VscodeDataOssPath)
	if err != nil {
		glog.Errorf("Save vscode server data failed. Vscode server: %+v. Error: %v", *s, err)
	}

	// Save the workspace data to oss.
	err = s.save(s.WorkspaceDir, s.WorkspaceOssPath)
	if err != nil {
		glog.Errorf("Save workspace data failed. Vscode server: %+v. Error: %v", *s, err)
	}
}

// load Load tar.gz from oss and extract to local directory.
// src The source oss object path.
// dst The destination local directory.
func (s *Server) load(src string, dst string) error {
	bucket, err := s.OssClient.Bucket(s.OssBucketName)
	if err != nil {
		glog.Errorf("Get oss bucket failed. Vscode server: %+v", *s)
		return err
	}

	body, err := bucket.GetObject(src)
	if err != nil {
		srvErr := err.(oss.ServiceError)
		if srvErr.Code == "NoSuchKey" && srvErr.StatusCode == 404 {
			// No workspace data. Just create workspace directory and return.
			if err = os.MkdirAll(dst, 0755); err != nil {
				glog.Errorf("Create local directory %s failed. Error: %v", dst, err)
				return err
			}
			return nil
		} else {
			glog.Errorf("Get oss object %s failed. Error: %v", src, err)
			return err
		}
	}
	err = tar.ExtractTarGz(body, dst)
	if err != nil {
		glog.Errorf("Extract tar gz failed. Local directory: %s Error: %v", dst, err)
		return err
	}
	glog.Infof("Load succeeded. Oss path: %s Local directory: %s", src, dst)
	return nil
}

// save Archive the local directory and save to the oss.
// src The source local directory.
// dst The destination oss object path.
func (s *Server) save(src string, dst string) error {
	bucket, err := s.OssClient.Bucket(s.OssBucketName)
	if err != nil {
		glog.Errorf("Get oss bucket failed. Vscode server: %+v Error: %v", *s, err)
		return err
	}

	buf := bytes.NewBuffer(nil)
	tar.TarGz(src, buf)
	err = bucket.PutObject(dst, buf)
	if err != nil {
		glog.Errorf("Put oss bucket %s failed. Error: %v", dst, err)
		return err
	}
	glog.Infof("Save succeeded. Local directory:%s Oss path: %s", src, dst)
	return nil
}
