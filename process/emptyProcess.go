package process

import "agentsc/data-proxy/common"

type EmptyProcess struct{}

func (p EmptyProcess) StartProcess() {
	for msg := range common.PreProcessChan {
		if (msg != nil) {
			common.PostProcessChan <- msg
		}
	}
}