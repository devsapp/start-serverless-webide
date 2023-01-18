package main

import (
	"aliyun/serverless/webide-server/pkg/context"
	"aliyun/serverless/webide-server/pkg/vscode"
	"flag"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/golang/glog"
	"github.com/spf13/viper"
)

type ServerManager struct {
	VscodeServer *vscode.Server         // backend vscode server
	Proxy        *httputil.ReverseProxy // frontend reverse proxy
}

// init implements the FC initializer instance lifecycle callback, called by FC runtime before processing the request.
func (sm *ServerManager) init() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		glog.Infof("Starting server manager init ...")

		// Get contextSource option from config file.
		// contextSource indicates where to get the context.
		// In FC runtime, context should be parsed from the request headers.
		// In VM or container, the context should be parsed from the environment variables.
		viper.SetDefault("contextSource", "fc")
		ctxSource := viper.GetString("contextSource")

		var err error
		var ctx *context.Context
		if ctxSource == "env" {
			ctx, err = context.NewFromEnvVars()
		} else {
			ctx, err = context.New(r)
		}
		if err != nil {
			glog.Errorf("Get context from %s failed. Error: %v", ctxSource, err)
			// Context failed because of invalid ak id, ak secret and security token, then return 403 Forbidden error.
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, err.Error())
			return
		}

		// Create the vscode server.
		sm.VscodeServer, err = vscode.NewServer(ctx)
		if err != nil {
			glog.Errorf("Create vscode server failed. Error: %v", err)
			// Create vscode server failed because of invalid ak id, ak secret and security token, then return 403 Forbidden error.
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, err.Error())
			return
		}

		// Create the reverse proxy.
		url, err := url.Parse("http://" + sm.VscodeServer.Host + ":" + sm.VscodeServer.Port)
		if err != nil {
			glog.Errorf("Parse url %s failed. Error: %v", sm.VscodeServer.Host, err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, err.Error())
			return
		}
		sm.Proxy = httputil.NewSingleHostReverseProxy(url)
		glog.Infof("Create reverse proxy succeeded. Url: %s", url)

		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "init handler success")
		glog.Infof("Server manager init success.")
	}
}

func (sm *ServerManager) shutdown() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		glog.Infof("Starting server manager shutdown ...")

		sm.VscodeServer.Shutdown()

		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "pre-stop handler success")
		glog.Infof("Server manager shutdown success.")
	}
}

func (sm *ServerManager) process() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r != nil {
			sm.Proxy.ServeHTTP(w, r)
		} else {
			glog.Errorf("The input request parameter is nil!")
		}
	}
}

func main() {
	flag.Parse()
	defer glog.Flush()

	// Get the directory of current running process.
	ex, err := os.Executable()
	if err != nil {
		glog.Fatalf("Failed to get the directory of current running process. Error: %v", err)
	}
	configDir := filepath.Dir(ex)
	configFile := filepath.Join(configDir, "config.yaml")

	// Setup the config file.
	viper.SetConfigFile(configFile)

	// Read the configurations from the specified file.
	err = viper.ReadInConfig()
	if err != nil {
		glog.Fatalf("Failed to read ide server config file. Error: %v", err)
	}
	glog.Infof("Reverse proxy read config file from directory: %s", configDir)

	sm := &ServerManager{}

	// Register the initializer handler.
	http.HandleFunc("/initialize", sm.init())

	// Register the shutdown handler.
	http.HandleFunc("/pre-stop", sm.shutdown())

	// Handle all other requests to your server using the proxy.
	http.HandleFunc("/", sm.process())

	// Start the proxy server.
	proxyServer := &http.Server{
		Addr:        ":9000",
		IdleTimeout: 5 * time.Minute,
	}

	glog.Infof("Reverse proxy listen at %s ...", proxyServer.Addr)
	glog.Fatalf("Reverse proxy run. Error: %v", proxyServer.ListenAndServe())
}
