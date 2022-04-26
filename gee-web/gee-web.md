# 7天用Go从零实现Web框架Gee教程

[toc]

## Day0

### 设计一个框架

大部分时候，我们需要实现一个 Web 应用，第一反应是应该使用哪个框架。不同的框架设计理念和提供的功能有很大的差别。比如 Python 语言的 `django`和`flask`，前者大而全，后者小而美。Go语言/golang 也是如此，新框架层出不穷，比如`Beego`，`Gin`，`Iris`等。那为什么不直接使用标准库，而必须使用框架呢？在设计一个框架之前，我们需要回答**框架核心为我们解决了什么问题**。只有理解了这一点，才能想明白我们需要在框架中实现什么功能。

我们先看看标准库net/http如何处理一个请求

```go
func main() {
    http.HandleFunc("/", handler)
    http.HandleFUnc("/count", counter)
    log.Fatal(http.ListenAndServe("localhost:8000", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "URL.Path = %q\n", r.URL.Path)
}
```

`net/http` 提供了基础的Web功能，即**监听端口，映射静态路由，解析HTTP报文**。一些Web开发中简单的需求并不支持，需要手工实现。

- **动态路由**：例如`hello/:name`，`hello/*`这类的规则。
- **鉴权**：没有分组/统一鉴权的能力，需要在每个路由映射的handler中实现。
- **模板**：没有统一简化的HTML机制。
- …

当我们离开框架，使用基础库时，需要频繁手工处理的地方，就是框架的价值所在。但并不是每一个频繁处理的地方都适合在框架中完成。Python有一个很著名的Web框架，名叫`bottle`，整个框架由`bottle.py`一个文件构成，共4400行，可以说是一个微框架。那么理解这个微框架提供的特性，可以帮助我们理解框架的核心能力。

- **路由(Routing)**：将请求映射到函数，支持动态路由。例如`'/hello/:name`。
- **模板(Templates)**：使用内置模板引擎提供模板渲染机制。
- **工具集(Utilites)**：提供对 cookies，headers 等处理机制。
- **插件(Plugin)**：Bottle本身功能有限，但提供了插件机制。可以选择安装到全局，也可以只针对某几个路由生效。
- …


### Gee 框架

这个教程将使用 Go 语言实现一个简单的 Web 框架，起名叫做`Gee`，`geektutu.com`的前三个字母。我第一次接触的 Go 语言的 Web 框架是`Gin`，`Gin`的代码总共是14K，其中测试代码9K，也就是说实际代码量只有5K。`Gin`也是我非常喜欢的一个框架，与Python中的`Flask`很像，小而美。

`7天实现Gee框架`这个教程的很多设计，包括源码，参考了`Gin`，大家可以看到很多Gin的影子。

时间关系，同时为了尽可能地简洁明了，这个框架中的很多部分实现的功能都很简单，但是尽可能地体现一个框架核心的设计原则。例如`Router`的设计，虽然支持的动态路由规则有限，但为了性能考虑匹配算法是用`Trie树`实现的，`Router`最重要的指标之一便是性能。

## Day1 HTTP基础

- 简单介绍`net/http`库以及`http.Handler`接口。
- 搭建`Gee`框架的雏形，**代码约50行**。

### 标准库启动Web服务

Go语言内置了 `net/http`库，封装了HTTP网络编程的基础的接口，我们实现的`Gee` Web 框架便是基于`net/http`的。我们接下来通过一个例子，简单介绍下这个库的使用。

day1-http-base/base1/main.go

我们设置了2个路由，`/`和`/hello`，分别绑定 `indexHandler` 和 `helloHandler` ， 根据不同的HTTP请求会调用不同的处理函数。访问`/`，响应是`URL.Path = /`，而`/hello`的响应则是请求头(header)中的键值对信息。

用 curl 这个工具测试一下，将会得到如下的结果。

```shell
yuancf1024@LAPTOP-22O3I9E3:~/7days-golang/gee-web/day1-http-base$ curl http://localhost:9999/
URL.Path = "/"
yuancf1024@LAPTOP-22O3I9E3:~/7days-golang/gee-web/day1-http-base$ curl http://localhost:9999/hello
Header["User-Agent"] = ["curl/7.68.0"]
Header["Accept"] = ["*/*"]
```

main 函数的最后一行，是用来启动 Web 服务的，第一个参数是地址，`:9999`表示在 9999 端口监听。而第二个参数则代表处理所有的HTTP请求的实例，`nil` 代表使用标准库中的实例处理。第二个参数，则是我们基于`net/http`标准库实现Web框架的入口。

### 实现http.Handler接口

```go
package http

type Handler interface {
    ServeHTTP(w ResponseWriter, r *Request)
}

func ListenAndServe(address string, h Handler) error
```

第二个参数的类型是什么呢？通过查看`net/http`的源码可以发现，`Handler`是一个接口，需要实现方法 ServeHTTP ，也就是说，**只要传入任何实现了 ServerHTTP 接口的实例，所有的HTTP请求，就都交给了该实例处理了**。马上来试一试吧。

day1-http-base/base2/main.go

