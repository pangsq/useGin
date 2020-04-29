# Gin router analysis 1

Gin是Golang开发的一个Web框架，特点是轻量、高效。

由于Golang本身net/http功能已经很强大且易用，一般小型Web项目用自带的库足矣。

Gin更多地是关注复杂的router逻辑和方便中间件的嵌入（如认证、日志等），以便于团队协作和代码的工程化管理。

本文来分析下Gin的router。

## 从一个普普通通的Gin使用例子开始

```
// r := gin.New()
r := gin.Default() // Default()会在New的基础上，注入默认的Logger和Recovery两个handler
rg := r.Group("/v1", middleware1, middleware2) // 创建一个RouterGroup，并注册两个middleware
rg1 := rg.Use(middleware2)  // 注册一个middlerware
rg1.Handle(http.MethodGet, "/ping", handler1)  // 注册一个request handler
r.Run()
```

先搞明白各种方法（除了Run）的返回类型。

1. Default/New 返回的是 *Engine
2. Group 返回的是 *RouterGroup
3. Use 返回的是 IRoutes
4. Handler 返回的是 IRoutes

### IRoutes

首先来看看出现的接口IRoutes，涵盖了各种http handler的注册。

Use用于中间件的注册，Handle是通用的handler注册接口（可以在入参中指定是httpMethod）,Any匹配任意httpMethod，从GET到HEAD是特定httpMethod的handler注册，而StaticFile/Static/StaticFS用于静态资源的注册。

```golang
type IRoutes interface {
	Use(...HandlerFunc) IRoutes

	Handle(string, string, ...HandlerFunc) IRoutes
	Any(string, ...HandlerFunc) IRoutes
	GET(string, ...HandlerFunc) IRoutes
	POST(string, ...HandlerFunc) IRoutes
	DELETE(string, ...HandlerFunc) IRoutes
	PATCH(string, ...HandlerFunc) IRoutes
	PUT(string, ...HandlerFunc) IRoutes
	OPTIONS(string, ...HandlerFunc) IRoutes
	HEAD(string, ...HandlerFunc) IRoutes

	StaticFile(string, string) IRoutes
	Static(string, string) IRoutes
	StaticFS(string, http.FileSystem) IRoutes
}
```

此处先引出IRouter接口，下文即将用到。

IRouter比起IRoutes仅多另一个Group方法。

```golang
type IRouter interface {
	IRoutes
	Group(string, ...HandlerFunc) *RouterGroup
}
```

### Engine

```golang
type Engine struct {
	RouterGroup

	... //其他暂不关心的成员
}
...
var _ IRouter = &Engine{}
```

Engine中与路由相关的是一个RouterGroup。按Golang匿名成员的写法，Engine继承了RouterGroup的方法，我们可以将Engine看做是功能更加丰富的RouterGroup。

### RouterGroup

```golang
type RouterGroup struct {
	Handlers HandlersChain
	basePath string
	engine   *Engine
	root     bool
}
...

var _ IRouter = &RouterGroup{}
```

RouterGroup是Gin路由功能的核心组件，它实现了IRouter/IRoutes接口。

在例子中Use/Handler返回的实际也是它，但通过IRoutes接口限制，避免了对Use/Handler结果再调用Group。

RouterGroup的成员：
1. Handlers HandlersChain  // 存放中间件方法的数组。由于中间件（如认证、日志等）本质也是和业务逻辑相同的HandlerFunc，所以最终会在调用业务逻辑的HandlerFunc前先调用这些HandlerFunc
2. basePath string         // 基础url路径
3. engine *Engine          // Engine是一个项目中唯一的，所有的RouterGroup都会指向同一个
4. root bool               // Engine本身也是个RouterGroup，它被认为是root

## 关于RouterGroup的一切

RouterGroup的设计并不复杂，在routergroup.go中完全实现。

### Use —— 中间件的添加

```golang
func (group *RouterGroup) Use(middleware ...HandlerFunc) IRoutes {
	group.Handlers = append(group.Handlers, middleware...)
	return group.returnObj()
}
```

将middleware添加到group.Handlers，并返回本身（如果是root，则返回engine）

### Group —— 创建子RouterGroup

```golang
func (group *RouterGroup) Group(relativePath string, handlers ...HandlerFunc) *RouterGroup {
	return &RouterGroup{
		Handlers: group.combineHandlers(handlers),
		basePath: group.calculateAbsolutePath(relativePath),
		engine:   group.engine,
	}
}
```

1. 创建新的RouterGroup
2. 新RouterGroup继承此时原RouterGroup的Handlers，并添加新的handlers。
    1. 此后若原RouterGroup继续使用Use添加handler，新RouterGroup的handlers不随之改变
    2. 此处的handlers均是middlewares
3. 计算新RouterGroup的basePath
4. engine依旧指向同一个engine

### handle —— 注册业务逻辑的handler

```golang
func (group *RouterGroup) handle(httpMethod, relativePath string, handlers HandlersChain) IRoutes {
	absolutePath := group.calculateAbsolutePath(relativePath)
	handlers = group.combineHandlers(handlers)
	group.engine.addRoute(httpMethod, absolutePath, handlers)
	return group.returnObj()
}
```

1. 计算url的绝对路径
2. 计算包含一些列middlewares和最终handler的handlers
3. 调用engine.addRoute将route注册
    1. 此后若group的Handlers有变化，也不对已经注册的这条路由有任何影响
4. 返回group

Handle/Any/Get/POST/DELETE/PATCH/PUT/OPTIONS/HEAD/StaticFile/Static/StaticFS的实现均基于handle。

挑一个比较特殊的Any看看

```golang
func (group *RouterGroup) Any(relativePath string, handlers ...HandlerFunc) IRoutes {
	group.handle(http.MethodGet, relativePath, handlers)
	group.handle(http.MethodPost, relativePath, handlers)
	group.handle(http.MethodPut, relativePath, handlers)
	group.handle(http.MethodPatch, relativePath, handlers)
	group.handle(http.MethodHead, relativePath, handlers)
	group.handle(http.MethodOptions, relativePath, handlers)
	group.handle(http.MethodDelete, relativePath, handlers)
	group.handle(http.MethodConnect, relativePath, handlers)
	group.handle(http.MethodTrace, relativePath, handlers)
	return group.returnObj()
}
```

很直接，将所有的httpMethod都注册。

## 总结

1. 通过New/Default创建Engine
2. Engine/RouterGroup存放basePath和handlers（用于中间件），使用Use可以往handlers中添加新的middleware
3. 基于Engine或RouterGroup可以创建新的RouterGroup，新的RouterGroup继承创建时Engine或RouterGroup的handlers
4. 使用Engine或RouterGroup的Handle/Any/Get/...等方法注册业务用handler，会使用其中包含的handlers（HandlersChain）和basePath
