package svc

import (
	"testing"

	"ai-gozero-agent/api/internal/utils"
)

type evalCase struct {
	question    string
	groundTruth string
	keywords    []string
}

type retrievalCase struct {
	query         string
	relevantTitle string
}

type evalDataset struct {
	title            string
	content          string
	cases            []evalCase
	retrievalQueries []string
}

var knowledgeEvalDatasets = []evalDataset{
	{
		title: "__rag_eval_knowledge_go_concurrency__",
		content: `
## goroutine 基本概念
goroutine 是 Go 语言的轻量级线程，由 Go 运行时调度。初始栈约 2KB，可动态扩缩。

## goroutine 与线程的区别
操作系统线程需要 1~8MB 固定栈，goroutine 更轻量。调度由用户态运行时完成，避免系统调用开销。

## channel 通信机制
goroutine 之间通过 channel 通信，遵循 CSP 模型。channel 分有缓冲和无缓冲两种。

## 内存模型
Go 的内存模型定义了 goroutine 之间可见性规则，依赖 happens-before 关系。

## GC 与 goroutine
Go 使用三色标记 GC，STW 时间极短，goroutine 感知不到大多数 GC 停顿。
`,
		cases: []evalCase{
			{"goroutine 和操作系统线程有什么区别？", "goroutine 是用户态轻量级线程，初始栈约 2KB 可动态扩缩，由 Go 运行时调度；操作系统线程需要 1~8MB 固定栈，调度涉及系统调用开销。", []string{"轻量", "2KB", "用户态", "运行时", "系统调用"}},
			{"Go 语言中 channel 是什么？", "channel 是 goroutine 之间通信的管道，遵循 CSP 模型，分有缓冲和无缓冲两种。", []string{"通信", "CSP", "有缓冲", "无缓冲"}},
			{"Go 调度器通常采用什么并发调度模型？", "Go 调度器通常采用 M:N 调度思想，由运行时在用户态将大量 goroutine 映射到较少的操作系统线程上。", []string{"M:N", "goroutine", "线程", "用户态", "运行时"}},
			{"goroutine 的栈是固定大小的吗？", "不是，goroutine 初始栈约 2KB，可根据需要动态扩缩。", []string{"初始栈", "2KB", "动态扩缩"}},
			{"有缓冲 channel 和无缓冲 channel 在分类上有什么区别？", "两者都是 Go 中 channel 的类型，前者带缓冲区，后者不带缓冲区。", []string{"有缓冲", "无缓冲", "channel", "缓冲区"}},
			{"Go 内存模型的作用是什么？", "Go 内存模型定义了 goroutine 之间的可见性规则，并依赖 happens-before 关系描述顺序。", []string{"内存模型", "可见性", "happens-before"}},
			{"Go 的 GC 使用什么核心标记方法？", "Go 使用三色标记 GC 来进行垃圾回收。", []string{"GC", "三色标记", "垃圾回收"}},
			{"STW 停顿对 goroutine 的影响大吗？", "STW 时间极短，goroutine 感知不到大多数 GC 停顿。", []string{"STW", "极短", "goroutine", "停顿"}},
		},
		retrievalQueries: []string{
			"goroutine 和操作系统线程有什么区别",
			"Go 调度器是什么模型",
			"channel 通信机制",
			"Go 内存模型",
			"GC 停顿对 goroutine 的影响",
		},
	},
	{
		title: "__rag_eval_knowledge_redis__",
		content: `
## Redis 数据结构
Redis 常用数据结构包括 String、Hash、List、Set 和 ZSet。ZSet 适合排行榜等需要排序的场景。

## 缓存异常问题
缓存穿透是请求访问根本不存在的数据；缓存击穿是热点 key 失效瞬间大量并发请求打到数据库；缓存雪崩是大量 key 同时过期导致后端压力骤增。

## 持久化机制
Redis 提供 RDB 和 AOF 两种持久化方式。RDB 适合做快照备份，AOF 记录写命令，数据恢复通常更完整。

## 过期淘汰与高可用
Redis 可为 key 设置过期时间，并通过淘汰策略在内存不足时回收数据。主从复制和哨兵机制可以提高高可用能力。

## 分布式锁
Redis 分布式锁通常使用 SET key value NX EX 的方式实现，要求原子设置并设置过期时间，防止死锁。
`,
		cases: []evalCase{
			{"Redis 常用数据结构有哪些？", "Redis 常用数据结构包括 String、Hash、List、Set 和 ZSet，其中 ZSet 适合排行榜等需要排序的场景。", []string{"String", "Hash", "List", "Set", "ZSet"}},
			{"Redis 的 ZSet 适合什么场景？", "ZSet 适合排行榜等需要按分数排序的场景。", []string{"ZSet", "排行榜", "排序"}},
			{"什么是缓存穿透？", "缓存穿透是请求访问根本不存在的数据，缓存和数据库都没有该数据。", []string{"缓存穿透", "不存在", "缓存", "数据库"}},
			{"缓存击穿和缓存雪崩有什么区别？", "缓存击穿是热点 key 失效瞬间大量请求打到数据库，缓存雪崩是大量 key 同时过期导致后端压力骤增。", []string{"击穿", "热点 key", "雪崩", "同时过期"}},
			{"Redis 的持久化方式有哪些？", "Redis 提供 RDB 和 AOF 两种持久化方式。", []string{"RDB", "AOF", "持久化"}},
			{"RDB 和 AOF 的主要区别是什么？", "RDB 是快照备份，AOF 记录写命令，数据恢复通常更完整。", []string{"RDB", "快照", "AOF", "写命令", "恢复"}},
			{"Redis 分布式锁通常怎么实现？", "Redis 分布式锁通常使用 SET key value NX EX 的方式实现，并设置过期时间防止死锁。", []string{"SET", "NX", "EX", "过期时间", "死锁"}},
			{"Redis 的主从复制和哨兵机制有什么作用？", "主从复制和哨兵机制用于提高 Redis 的高可用能力。", []string{"主从复制", "哨兵", "高可用"}},
		},
		retrievalQueries: []string{
			"Redis 常用数据结构",
			"缓存穿透是什么",
			"缓存击穿和雪崩区别",
			"RDB 和 AOF 的区别",
			"Redis 分布式锁怎么实现",
		},
	},
	{
		title: "__rag_eval_knowledge_mysql__",
		content: `
## 索引结构
MySQL 常用索引底层通常采用 B+ 树。B+ 树适合范围查询和磁盘存储场景，因为叶子节点有序且能够减少磁盘 I/O。

## 聚簇索引与二级索引
InnoDB 的主键索引通常是聚簇索引，叶子节点直接存放行数据。二级索引叶子节点保存主键值，需要回表才能读取完整行。

## 最左前缀原则
联合索引遵循最左前缀原则，查询条件从索引最左列开始连续匹配时更容易命中索引。

## 事务特性与隔离级别
事务具备 ACID 四大特性，包括原子性、一致性、隔离性和持久性。常见隔离级别包括读未提交、读已提交、可重复读和串行化。

## MVCC
MVCC 通过多版本并发控制减少读写冲突，使读操作在很多情况下无需加锁即可读取历史版本数据。
`,
		cases: []evalCase{
			{"MySQL 常用索引底层通常采用什么结构？", "MySQL 常用索引底层通常采用 B+ 树，因为它适合范围查询和磁盘存储场景。", []string{"B+ 树", "范围查询", "磁盘"}},
			{"为什么 B+ 树适合数据库索引？", "因为 B+ 树叶子节点有序，适合范围查询，并且能够减少磁盘 I/O。", []string{"叶子节点有序", "范围查询", "磁盘 I/O"}},
			{"聚簇索引和二级索引有什么区别？", "聚簇索引叶子节点直接存放行数据，二级索引叶子节点保存主键值，查询完整行通常需要回表。", []string{"聚簇索引", "二级索引", "行数据", "主键值", "回表"}},
			{"什么是最左前缀原则？", "联合索引遵循最左前缀原则，查询条件从最左列开始连续匹配时更容易命中索引。", []string{"联合索引", "最左前缀", "最左列", "连续匹配"}},
			{"事务的 ACID 特性包括哪些内容？", "事务的 ACID 特性包括原子性、一致性、隔离性和持久性。", []string{"原子性", "一致性", "隔离性", "持久性"}},
			{"MySQL 常见的事务隔离级别有哪些？", "常见隔离级别包括读未提交、读已提交、可重复读和串行化。", []string{"读未提交", "读已提交", "可重复读", "串行化"}},
			{"MVCC 的作用是什么？", "MVCC 通过多版本并发控制减少读写冲突，使读操作在很多情况下无需加锁即可读取历史版本数据。", []string{"MVCC", "多版本", "读写冲突", "无需加锁"}},
			{"二级索引为什么可能需要回表？", "因为二级索引叶子节点保存的是主键值，不是完整行数据，所以读取完整行时可能需要根据主键再查一次表。", []string{"二级索引", "主键值", "完整行数据", "回表"}},
		},
		retrievalQueries: []string{
			"MySQL B+ 树索引",
			"聚簇索引和二级索引区别",
			"最左前缀原则",
			"事务 ACID 特性",
			"MVCC 有什么作用",
		},
	},
	{
		title: "__rag_eval_knowledge_microservice__",
		content: `
## 网关与服务拆分
微服务架构通常会按业务能力拆分服务，并通过 API 网关统一接入、鉴权和路由。

## 服务注册与发现
服务注册与发现机制用于让服务实例动态上线下线时仍能被其他服务找到，常见实现方式包括注册中心。

## 熔断、重试与幂等
在分布式调用中，熔断用于防止故障扩散，重试用于应对瞬时失败，幂等用于防止重复请求造成副作用。

## 异步消息
Kafka、RabbitMQ 等消息队列适合异步解耦、削峰填谷和最终一致性处理。

## 分布式事务
分布式事务场景中，很多系统会采用最终一致性方案，而不是强一致的全局事务。
`,
		cases: []evalCase{
			{"微服务架构中 API 网关的作用是什么？", "API 网关用于统一接入、鉴权和路由。", []string{"API 网关", "统一接入", "鉴权", "路由"}},
			{"为什么微服务需要服务注册与发现？", "服务注册与发现用于让服务实例动态上线下线时仍能被其他服务找到。", []string{"服务注册", "服务发现", "动态上线", "实例"}},
			{"熔断的作用是什么？", "熔断用于防止分布式调用中的故障继续扩散。", []string{"熔断", "故障扩散", "分布式调用"}},
			{"为什么在分布式系统中需要幂等设计？", "幂等用于防止重复请求造成重复副作用。", []string{"幂等", "重复请求", "副作用"}},
			{"消息队列在微服务中常见的作用有哪些？", "Kafka、RabbitMQ 等消息队列常用于异步解耦、削峰填谷和最终一致性处理。", []string{"异步解耦", "削峰填谷", "最终一致性", "Kafka", "RabbitMQ"}},
			{"重试机制主要解决什么问题？", "重试机制主要用于应对瞬时失败。", []string{"重试", "瞬时失败"}},
			{"为什么很多分布式事务场景采用最终一致性？", "因为很多系统在分布式事务场景中更倾向于采用最终一致性，而不是强一致的全局事务。", []string{"分布式事务", "最终一致性", "强一致"}},
			{"微服务为什么要按业务能力拆分服务？", "微服务架构通常会按业务能力拆分服务，以实现更清晰的职责划分。", []string{"业务能力", "拆分服务", "职责划分"}},
		},
		retrievalQueries: []string{
			"API 网关的作用",
			"服务注册与发现为什么需要",
			"熔断和幂等的作用",
			"消息队列的作用",
			"最终一致性是什么",
		},
	},
	{
		title: "__rag_eval_knowledge_kubernetes__",
		content: `
## Docker 镜像与容器
Docker 镜像是静态模板，容器是镜像运行后的实例。

## Pod、Deployment 与 Service
Kubernetes 中 Pod 是最小调度单元，Deployment 负责管理副本和滚动更新，Service 提供稳定访问入口。

## 健康检查与滚动更新
Readiness Probe 用于判断服务是否可接收流量，Liveness Probe 用于判断容器是否需要重启。滚动更新可以在发布新版本时减少停机时间。

## 配置管理
ConfigMap 通常用于保存普通配置，Secret 用于保存敏感信息。

## CI/CD 与可观测性
CI/CD 用于自动化构建、测试与发布；Prometheus 和 Grafana 常用于监控与可视化。
`,
		cases: []evalCase{
			{"Docker 镜像和容器有什么区别？", "Docker 镜像是静态模板，容器是镜像运行后的实例。", []string{"镜像", "静态模板", "容器", "实例"}},
			{"Kubernetes 中 Pod 的作用是什么？", "Pod 是 Kubernetes 中最小的调度单元。", []string{"Pod", "最小调度单元"}},
			{"Deployment 在 Kubernetes 中主要负责什么？", "Deployment 主要负责管理副本和滚动更新。", []string{"Deployment", "副本", "滚动更新"}},
			{"Service 在 Kubernetes 中起什么作用？", "Service 提供稳定访问入口。", []string{"Service", "稳定访问入口"}},
			{"Readiness Probe 和 Liveness Probe 的区别是什么？", "Readiness Probe 判断服务是否可接收流量，Liveness Probe 判断容器是否需要重启。", []string{"Readiness", "可接收流量", "Liveness", "重启"}},
			{"为什么滚动更新能够减少停机时间？", "因为滚动更新在发布新版本时逐步替换实例，从而减少停机时间。", []string{"滚动更新", "新版本", "逐步替换", "减少停机时间"}},
			{"ConfigMap 和 Secret 的区别是什么？", "ConfigMap 通常保存普通配置，Secret 用于保存敏感信息。", []string{"ConfigMap", "普通配置", "Secret", "敏感信息"}},
			{"CI/CD 和 Prometheus、Grafana 分别解决什么问题？", "CI/CD 用于自动化构建、测试与发布，Prometheus 和 Grafana 常用于监控与可视化。", []string{"CI/CD", "构建", "测试", "发布", "监控", "可视化"}},
		},
		retrievalQueries: []string{
			"Docker 镜像和容器区别",
			"Pod Deployment Service 关系",
			"Readiness 和 Liveness 区别",
			"ConfigMap 和 Secret 区别",
			"Prometheus Grafana 做什么",
		},
	},
}

var resumeEvalDatasets = []evalDataset{
	{
		title: "__rag_eval_resume_zhangsan__",
		content: `张三，5年Go开发经验。
曾在字节跳动负责推荐系统后端开发，主导了用户行为数据处理管道的重构，将吞吐量提升了3倍。
熟悉分布式系统设计，了解Raft共识算法原理。
业余时间喜欢打篮球，曾获校级三分球比赛冠军。
目前关注云原生方向，正在学习Kubernetes Operator开发。`,
		cases: []evalCase{
			{"张三有哪些工作经历？", "张三曾在字节跳动负责推荐系统后端开发，主导用户行为数据处理管道重构。", []string{"张三", "字节跳动", "推荐系统", "后端开发", "管道重构"}},
			{"张三负责的项目把吞吐量提升了多少倍？", "张三主导的用户行为数据处理管道重构将吞吐量提升了 3 倍。", []string{"张三", "吞吐量", "3倍", "重构"}},
			{"张三熟悉哪些技术方向？", "张三熟悉分布式系统设计，了解 Raft 共识算法。", []string{"张三", "分布式", "Raft", "共识算法"}},
			{"张三目前正在学习什么？", "张三目前关注云原生方向，正在学习 Kubernetes Operator 开发。", []string{"张三", "云原生", "Kubernetes", "Operator"}},
			{"张三的业余爱好是什么？", "张三业余时间喜欢打篮球，曾获校级三分球比赛冠军。", []string{"张三", "篮球", "三分球", "冠军"}},
			{"张三了解哪种共识算法？", "张三了解 Raft 共识算法原理。", []string{"张三", "Raft", "共识算法"}},
		},
		retrievalQueries: []string{
			"张三有哪些工作经历",
			"张三吞吐量提升了多少倍",
			"张三正在学习什么",
			"张三了解哪种共识算法",
		},
	},
	{
		title: "__rag_eval_resume_lisi__",
		content: `李四，4年Java开发经验。
曾在美团负责订单中心与促销结算服务开发，推动热点接口缓存改造，使核心接口峰值QPS提升了2倍。
熟悉Spring Boot、Spring Cloud、MySQL和Kafka，了解分布式事务中的Seata方案。
业余时间喜欢跑步，完成过两次半程马拉松。
目前正在学习Service Mesh与Istio治理能力。`,
		cases: []evalCase{
			{"李四有哪些工作经历？", "李四曾在美团负责订单中心与促销结算服务开发。", []string{"李四", "美团", "订单中心", "促销结算"}},
			{"李四负责的改造使核心接口峰值 QPS 提升了多少？", "李四推动热点接口缓存改造，使核心接口峰值 QPS 提升了 2 倍。", []string{"李四", "缓存改造", "QPS", "2倍"}},
			{"李四熟悉哪些技术栈？", "李四熟悉 Spring Boot、Spring Cloud、MySQL 和 Kafka。", []string{"李四", "Spring Boot", "Spring Cloud", "MySQL", "Kafka"}},
			{"李四了解哪种分布式事务方案？", "李四了解分布式事务中的 Seata 方案。", []string{"李四", "Seata", "分布式事务"}},
			{"李四的业余爱好是什么？", "李四业余时间喜欢跑步，完成过两次半程马拉松。", []string{"李四", "跑步", "半程马拉松"}},
			{"李四目前正在学习什么？", "李四目前正在学习 Service Mesh 与 Istio 治理能力。", []string{"李四", "Service Mesh", "Istio", "治理"}},
		},
		retrievalQueries: []string{
			"李四有哪些工作经历",
			"李四峰值QPS提升了多少",
			"李四熟悉哪些技术栈",
			"李四目前学习什么",
		},
	},
	{
		title: "__rag_eval_resume_wangmin__",
		content: `王敏，3年前端开发经验。
曾在小红书参与社区增长平台前端开发，主导设计系统组件重构，使核心页面首屏渲染时间下降了35%。
熟悉React、TypeScript、Vite和前端性能优化，了解BFF架构设计。
业余时间喜欢摄影，长期维护个人旅行摄影博客。
目前正在学习WebAssembly与前端工程化构建优化。`,
		cases: []evalCase{
			{"王敏有哪些工作经历？", "王敏曾在小红书参与社区增长平台前端开发。", []string{"王敏", "小红书", "社区增长", "前端开发"}},
			{"王敏主导的重构带来了什么性能收益？", "王敏主导设计系统组件重构，使核心页面首屏渲染时间下降了 35%。", []string{"王敏", "设计系统", "组件重构", "首屏渲染", "35%"}},
			{"王敏熟悉哪些技术？", "王敏熟悉 React、TypeScript、Vite 和前端性能优化。", []string{"王敏", "React", "TypeScript", "Vite", "性能优化"}},
			{"王敏了解什么架构设计？", "王敏了解 BFF 架构设计。", []string{"王敏", "BFF", "架构设计"}},
			{"王敏的业余爱好是什么？", "王敏业余时间喜欢摄影，长期维护个人旅行摄影博客。", []string{"王敏", "摄影", "博客"}},
			{"王敏目前正在学习什么？", "王敏目前正在学习 WebAssembly 与前端工程化构建优化。", []string{"王敏", "WebAssembly", "前端工程化", "构建优化"}},
		},
		retrievalQueries: []string{
			"王敏有哪些工作经历",
			"王敏的性能优化成果是什么",
			"王敏了解什么架构",
			"王敏现在学什么",
		},
	},
	{
		title: "__rag_eval_resume_zhaolei__",
		content: `赵磊，6年运维与云原生平台经验。
曾在阿里云负责Kubernetes集群平台运维与发布流水线改造，将核心服务平均发布时间从20分钟缩短到5分钟。
熟悉Docker、Kubernetes、Prometheus、Grafana和CI/CD流程，了解混沌工程实践。
业余时间喜欢徒步，每年都会参加长距离山地穿越活动。
目前正在学习平台工程与多集群治理方案。`,
		cases: []evalCase{
			{"赵磊有哪些工作经历？", "赵磊曾在阿里云负责 Kubernetes 集群平台运维与发布流水线改造。", []string{"赵磊", "阿里云", "Kubernetes", "运维", "发布流水线"}},
			{"赵磊的改造把平均发布时间缩短到了多少？", "赵磊将核心服务平均发布时间从 20 分钟缩短到 5 分钟。", []string{"赵磊", "20分钟", "5分钟", "发布时间"}},
			{"赵磊熟悉哪些技术方向？", "赵磊熟悉 Docker、Kubernetes、Prometheus、Grafana 和 CI/CD 流程。", []string{"赵磊", "Docker", "Kubernetes", "Prometheus", "Grafana", "CI/CD"}},
			{"赵磊了解什么实践方法？", "赵磊了解混沌工程实践。", []string{"赵磊", "混沌工程"}},
			{"赵磊的业余爱好是什么？", "赵磊业余时间喜欢徒步，每年都会参加长距离山地穿越活动。", []string{"赵磊", "徒步", "山地穿越"}},
			{"赵磊目前正在学习什么？", "赵磊目前正在学习平台工程与多集群治理方案。", []string{"赵磊", "平台工程", "多集群治理"}},
		},
		retrievalQueries: []string{
			"赵磊有哪些工作经历",
			"赵磊发布时间缩短到了多少",
			"赵磊熟悉哪些技术方向",
			"赵磊目前在学什么",
		},
	},
	{
		title: "__rag_eval_resume_chenchen__",
		content: `陈晨，4年数据与AI工程经验。
曾在百度参与企业知识问答平台建设，负责文本清洗、向量检索链路和RAG服务封装，使答案首条命中率提升了18%。
熟悉Python、Go、向量检索、Prompt设计和NLP应用，了解Transformer模型原理。
业余时间喜欢阅读科幻小说，并运营一个AI技术笔记专栏。
目前正在学习多智能体协作与推理编排框架。`,
		cases: []evalCase{
			{"陈晨有哪些工作经历？", "陈晨曾在百度参与企业知识问答平台建设，负责文本清洗、向量检索链路和 RAG 服务封装。", []string{"陈晨", "百度", "知识问答平台", "向量检索", "RAG"}},
			{"陈晨负责的项目把什么指标提升了多少？", "陈晨负责的知识问答平台建设使答案首条命中率提升了 18%。", []string{"陈晨", "首条命中率", "18%"}},
			{"陈晨熟悉哪些技术？", "陈晨熟悉 Python、Go、向量检索、Prompt 设计和 NLP 应用。", []string{"陈晨", "Python", "Go", "向量检索", "Prompt", "NLP"}},
			{"陈晨了解什么模型原理？", "陈晨了解 Transformer 模型原理。", []string{"陈晨", "Transformer", "模型原理"}},
			{"陈晨的业余爱好是什么？", "陈晨业余时间喜欢阅读科幻小说，并运营一个 AI 技术笔记专栏。", []string{"陈晨", "科幻小说", "AI技术笔记专栏"}},
			{"陈晨目前正在学习什么？", "陈晨目前正在学习多智能体协作与推理编排框架。", []string{"陈晨", "多智能体", "推理编排框架"}},
		},
		retrievalQueries: []string{
			"陈晨有哪些工作经历",
			"陈晨提升了什么指标",
			"陈晨熟悉哪些技术",
			"陈晨现在学习什么",
		},
	},
}

func flattenDatasetCases(datasets []evalDataset) []evalCase {
	var cases []evalCase
	for _, dataset := range datasets {
		cases = append(cases, dataset.cases...)
	}
	return cases
}

func flattenRetrievalCases(datasets []evalDataset) []retrievalCase {
	var cases []retrievalCase
	for _, dataset := range datasets {
		for _, query := range dataset.retrievalQueries {
			cases = append(cases, retrievalCase{
				query:         query,
				relevantTitle: dataset.title,
			})
		}
	}
	return cases
}

func datasetTitles(datasets []evalDataset) []string {
	titles := make([]string, 0, len(datasets))
	for _, dataset := range datasets {
		titles = append(titles, dataset.title)
	}
	return titles
}

func saveDatasetsWithSplitter(t *testing.T, store *VectorStore, datasets []evalDataset, splitter utils.Splitter) error {
	t.Helper()
	for _, dataset := range datasets {
		if err := store.SaveKnowledge(dataset.title, dataset.content, testCfg, splitter); err != nil {
			return err
		}
	}
	return nil
}

func cleanupKnowledgeTitles(t *testing.T, store *VectorStore, titles ...string) {
	t.Helper()
	for _, title := range titles {
		cleanupKnowledgeTitle(t, store, title)
	}
}
