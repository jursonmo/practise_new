
### suricata ips 实现方式
1. suricata 本身是有ips 的模式，原理是netfilter 截取数据传到应用层，应用层决定是否放行这个报文，但是这种方式的可能性能一般，流量大的时候或者设备的性能不高的场景下，可能会导致流量的转发效率低下以及更高延迟, 而且如果suricata异常，会影响上网。（这个方式的优点是每个数据报文都需要suricata 决策放行后才可以通过，即如果匹配到drop，就丢弃报文，一个都不会放行。）

2. 第二种实现方式，可以称之为旁路模式, suricata 通过af-packet 或者 pcap 读取网卡的数据（不影响数据继续走协议栈），数据如果匹配了suricata 的 drop rules 后会产生的日志，读取eve.json日志里五元组信息，然后根据五元组信息在 netfilter 链表里添加相应的iptalbes 规则进行阻断，这个种方式优点是对流量的转发影响较低，缺点是只能阻断告警之后的攻击。为了避免iptables 规则越加越多，影响路由器的转发效率，可以给iptables 规则设置一个截止时间(这个截止时间最好能是可配置)，比如一个小时，即一个小时后自动把对应的iptables 规则删除。这种做法适合阻断行为较少的场景。（你帮我们实现这种方式的ips）
