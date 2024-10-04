
zinx client 需要自己管控 连接的状态， 连接断开后，需要重新连接。 不够友好， 改天提pr 给zinx 完善下。

TOOD：

启动工作池的问题：
1. case ziface.IFuncRequest: // Internal function call request (内部函数调用request) 是什么情况下使用的？
2. 

func (mh *MsgHandle) StartOneWorker(workerID int, taskQueue chan ziface.IRequest) {
	zlog.Ins().DebugF("Worker ID = %d is started.", workerID)
	// Continuously wait for messages in the queue
	// (不断地等待队列中的消息)
	for {
		select {
		// If there is a message, take out the Request from the queue and execute the bound business method
		// (有消息则取出队列的Request，并执行绑定的业务方法)
		case request := <-taskQueue:

			switch req := request.(type) {

			case ziface.IFuncRequest:
				// Internal function call request (内部函数调用request)

				mh.doFuncHandler(req, workerID)

			case ziface.IRequest: // Client message request

				if !zconf.GlobalObject.RouterSlicesMode {
					mh.doMsgHandler(req, workerID)
				} else if zconf.GlobalObject.RouterSlicesMode {
					mh.doMsgHandlerSlices(req, workerID)
				}
			}
		}
	}
}
