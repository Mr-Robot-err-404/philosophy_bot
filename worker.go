package main

func receiveJobs(jobs <-chan HookPayload) {
	for task := range jobs {
		if task.Err != nil {
			continue
		}
	}
}
