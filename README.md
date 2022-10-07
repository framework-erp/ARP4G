# ARP4G
ARP4G是一个go语言实现的简化应用开发的框架。

通过 **A**ggregate（聚合）、**R**epository（仓库）、**P**rocess（过程）3个概念，隔离业务逻辑和技术实现细节，使开发者专注于产品业务本身。

## 一个简单的例子

```go
func (serv *OrderService) CompleteOrder(ctx context.Context, orderId string) *Order {
	//从仓库取出order
	order, _ := serv.orderRepository.Take(ctx, orderId)
	if order.state == "ongoing" {
		//改变他的状态
		order.state = "compleated"
		//返回改变后的order
		return order
	}
	return nil
}

type OrderService struct {
	orderRepository OrderRepository
}

type OrderRepository interface {
	Take(ctx context.Context, id any) (*Order, bool)
}
```

这里我们首先从订单仓库取出了一个订单（聚合），随后改变了他的状态，变成“已完成”，最后返回了这个“已完成”的订单。

这是一段业务代码，在这过程我们不关心查询和保存这些和数据库打交道的事情，我们也不关心 “并发改变订单状态所带来的问题” 这样的复杂技术细节，代码中没有任何技术细节，只有业务。

我们需要做的仅仅是调用该业务方法时用ARP4G包装一下（无需侵入这段纯粹的业务代码），ARP4G就会为你照顾一切技术细节。

订单仓库 “orderRepository” 不需要开发，可以使用[ARP4G的不同实现](#ARP4G的不同实现)来实现

## 安装

1. 首先需要 [Go](https://golang.org/) 已安装（**1.18及以上版本**）， 然后可以用以下命令安装ARP4G。

```sh
go get -u github.com/zhengchengdong/ARP4G
```

2. 在你的代码中 import：

```go
import "github.com/zhengchengdong/ARP4G"
```
## 快速开始
```go
package main

import (
	"context"
	"fmt"

	"github.com/zhengchengdong/ARP4G/arp"
)

func main() {
	greetingService := &GreetingService{}
	arp.Go(context.Background(), func(ctx context.Context) {
		//调用业务方法
		greetingService.SayHello(ctx)
	})
}

type GreetingService struct {
}

func (serv *GreetingService) SayHello(ctx context.Context) {
	fmt.Println("hello world")
}

```
## ARP4G的不同实现
### MongoDB
[ARP4G-mongodb](https://github.com/zhengchengdong/ARP4G-mongodb)
### Redis
[ARP4G-redis](https://github.com/zhengchengdong/ARP4G-redis)
## 使用ARP开发业务简介
假设有一段**完成订单**的业务，根据订单id找到相关订单，把它的状态改为已完成。以下将介绍如何使用**A**ggregate、**R**epository、**P**rocess3个概念且利用ARP4G框架完成这段简单的业务。
### Aggregate（聚合）
一个聚合就是表示某个实体的对象，该实体可能还包含别的实体，通常，这在业务上意味着被包含的实体是聚合的一部分，比如一辆**汽车**包含一个**引擎**。这里说的聚合就是[DDD_Aggregate](https://martinfowler.com/bliki/DDD_Aggregate.html)。

显然这里有一个聚合，Order
```go
type Order struct {
	Id    string
	State string
}
```
### Repository（仓库）
仓库，就是存放聚合对象的仓库。

你可以把它想象成一个真实世界的仓库。例如有个品牌汽车专卖店，它有一个**汽车仓库**停满了汽车，当然它不会有**引擎仓库**，因为当你从仓库开出一辆汽车的时候就已经包含了它的引擎了。
### 从仓库获取我要的聚合
### 完成我的业务逻辑
### ARP4G做了什么
在这之前我愿介绍一些业务设计的规则：

