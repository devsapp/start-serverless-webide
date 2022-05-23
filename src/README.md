# serverless-webide 帮助文档
<p align="center" class="flex justify-center">
    <a href="https://www.serverless-devs.com" class="ml-1">
    <img src="http://editor.devsapp.cn/icon?package=start-serverless-webide&type=packageType">
  </a>
  <a href="http://www.devsapp.cn/details.html?name=start-serverless-webide" class="ml-1">
    <img src="http://editor.devsapp.cn/icon?package=start-serverless-webide&type=packageVersion">
  </a>
  <a href="http://www.devsapp.cn/details.html?name=start-serverless-webide" class="ml-1">
    <img src="http://editor.devsapp.cn/icon?package=start-serverless-webide&type=packageDownload">
  </a>
</p>

<description>

> ***快速部署 Serverless VSCode webide 应用到阿里云函数计算***

</description>
<table>
</table>
<codepre id="codepre">
</codepre>
<deploy>

## 部署 & 体验

开通阿里云[函数计算](https://fcnext.console.aliyun.com/)，[对象存储](https://oss.console.aliyun.com)服务

<appcenter>

- :fire: 通过 [Serverless 应用中心](https://fcnext.console.aliyun.com/applications/create?template=start-serverless-webide) ，[![Deploy with Severless Devs](https://img.alicdn.com/imgextra/i1/O1CN01w5RFbX1v45s8TIXPz_!!6000000006118-55-tps-95-28.svg)](https://fcnext.console.aliyun.com/applications/create?template=start-serverless-webide)  该应用。 

</appcenter>

- 通过 [Serverless Devs Cli](https://www.serverless-devs.com/serverless-devs/install) 进行部署：
    - [安装 Serverless Devs Cli 开发者工具](https://www.serverless-devs.com/serverless-devs/install) ，并进行[授权信息配置](https://www.serverless-devs.com/fc/config) ；
    - 初始化项目：`s init start-serverless-webide -d serverless-webide`   
    - 进入项目，并进行项目部署：`cd serverless-webide && s deploy -y`

</deploy>
在浏览器中访问服务的网址。Web IDE 的配置以及 /workspace 下的数据将自动保存。

# 应用详情

基于 Serverless 架构和 Vscode 的**即开即用，用完即走**的轻量 Web IDE 服务。主要特点：

* 全功能 Vscode Web IDE，支持海量的插件。
* 虚拟机级别的多租安全隔离。
* 数据实时保存。用户可以随时关闭页面而不必担心数据丢失。
* 状态实时恢复。依托于函数计算极致的启动速度，秒级恢复到上次的状态。用户可随时继续。
* 资源利用率高，低成本。绝大多数 IDE 的使用是碎片化的，只在一天中的少部分时间被使用，因此 IDE 实例常驻是不明智的。借助函数计算完全按需付费，忙闲时单独定价的计费策略，成本比常驻型 IDE 实例低 3-10x。

# 开发
## 基本流程

本项目主要实现了一个 Reverse Proxy，请求的处理流程如下图所示。

![图片.png](https://cdn.nlark.com/yuque/0/2022/png/995498/1652601830486-bfea1122-433a-49d6-b276-02ab522d8b1e.png)

## 环境配置

在 `configs` 目录下，包含了一些配置文件。请根据需要修改对应的配置文件。

* `dev.yaml`：在本地启动 webide-server 所需的配置文件
* `test.yaml`：运行测试所需的配置文件
* `fc.yaml`：在函数计算（FC） runtime 环境中运行 webide-server 所需的配置文件

在本地启动 webide-server，或者运行测试，还需要配置以下3个环境变量。

* ALI_KEY_ID：您的阿里云 access key id
* ALI_KEY_SECRET：您的阿里云 access key secret
* ALI_REGION：您要运行测试的阿里云区域，例如 cn-hangzhou，cn-beijing 等等

## 开发调试

本地需要提前安装好 Golang, 下面的开发调试流程仅针对 mac 和 linux

> 如果您使用 windows 进行开发，且本地安装了 docker， 可以使用如下命令进入 linux 容器，完成下面的开发调试
> `docker run -it -v {your repo dir}:/code -p 9000:9000 golang bash`
> 其中 {your repo dir} 对应你 git clone 这个仓库的目录，进入容器后， 跳转到 /code 目录 

在项目根目录下按如下步骤执行 shell 命令。

1. 修改 `dev.yaml` 中的配置项，执行下述命令编译项目。成功后，会在项目根目录下新建 target 目录，包含了二进制文件，对应的启动配置文件等交付物。
   > **注意** `binaryDirectory` 这个值,  跟您的开发平台有有关， 您可以先 `make build` 之后， 参考 `third_party ` 下面的 `openvscode-server-v${VSCODE_SERVER_VERSION}-${OS}-${ARCH}` 目录， 然后修改 dev.yaml， 重新 `make build`

   ```shell
   make build
   ```

2. 在本地环境启动 webide server。

   ```shell
   ./target/webide-server -logtostderr=true
   ```

   > 如果您是 mac amd64 机型，./target/webide-server -logtostderr=true 有 readlink: illegal option -- f, 请尝试安装 `brew install coreutils` 解决，[详情](https://www.cnblogs.com/cphmvp/p/7070941.html) 

3. 请注意，step 2 只是创建了反向代理 ide-server，后台的 vscode-server 并没有启动。只有执行下述命令后，web ide 才功能就绪。其中端口请后 ide-server 启动时的端口保持一致。

   ```shell
   curl localhost:9000/initialize
   ```

4. Shutdown webide-server，将 vscode-server 的配置数据和 workspace 下的用户数据保存到 oss。

   ```shell
   curl localhost:9000/shutdown
   ```

## 本地测试

在本地运行测试，需要配置以下3个环境变量，以及 `configs` 目录中的 `test.yaml` 中的配置项。

* ALI_KEY_ID：您的阿里云 access key id
* ALI_KEY_SECRET：您的阿里云 access key secret
* ALI_REGION：您要运行测试的阿里云区域，例如 cn-hangzhou，cn-beijing 等等

在项目根目录执行命令运行测试。

```shell
make test
```

## 部署到 FC
### 1. 安装 Serverless Devs 工具

该项目使用 [Serverless Devs](https://docs.serverless-devs.com/serverless-devs/quick_start) 工具部署 FC 应用，请按照文档安装该工具。

### 2. 部署应用到函数计算（FC）
1. 修改 `fc.yaml` 中的配置项, 主要是 `ossBucketName` 这个值， 也可以通过 env `OSS_BUCKET_NAME` 来设置。
   > bucket 最好和函数是相同的 region

2. 使用 Serverless Devs 工具部署到 FC。

   ```shell
   s deploy
   ```

## 函数计算（FC）应用调试技巧

### 实时日志查询

可在 FC 控制台可查看函数实时日志。也可使用 Serverless Devs 工具查询实时日志。在项目根目录（s.yaml 所在目录）执行命令：

```shell
s logs --tail
```

### 实例登录

可在 FC 控制台登录实例。也可使用 Serverless Devs 工具登录。在项目根目录（s.yaml 所在目录）执行命令：

1. 首先列出当前函数的实例。

   ```shell
   s instance list
   ```

2. 然后登录实例。请将 `your-instance-id` 换成您在 step 1 中列出的实例 id。

   ```shell
   s instance exec -it your-instance-id /bin/bash
   ```

## 开发者社区

您如果有关于错误的反馈或者未来的期待，您可以在 [Serverless Devs repo Issues](https://github.com/serverless-devs/serverless-devs/issues) 中进行反馈和交流。如果您想要加入我们的讨论组或者了解 FC 组件的最新动态，您可以通过以下渠道进行：

<p align="center">

| <img src="https://serverless-article-picture.oss-cn-hangzhou.aliyuncs.com/1635407298906_20211028074819117230.png" width="130px" > | <img src="https://serverless-article-picture.oss-cn-hangzhou.aliyuncs.com/1635407044136_20211028074404326599.png" width="130px" > | <img src="https://serverless-article-picture.oss-cn-hangzhou.aliyuncs.com/1635407252200_20211028074732517533.png" width="130px" > |
|--- | --- | --- |
| <center>微信公众号：\`serverless\`</center> | <center>微信小助手：\`xiaojiangwh\`</center> | <center>钉钉交流群：\`33947367\`</center> | 

</p>

</devgroup>