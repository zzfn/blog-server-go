package tasks

import (
	"fmt"
	"github.com/go-co-op/gocron"
	"time"
)

func task() {
	fmt.Println("I am runnning task.")
}
func StartCronJobs() func() {
	scheduler := gocron.NewScheduler(time.UTC)

	// 示例任务: 每隔10秒输出文本
	//scheduler.Every(10).Seconds().Do(func() {
	//	fmt.Println("Running a task every 10 seconds.")
	//})
	//
	//scheduler.Every(2).Seconds().Do(func() {
	//	fmt.Println("Running a task every 2 seconds.")
	//})

	// 每天的特定时间执行任务
	scheduler.Cron("0 18 * * *").Do(task)
	scheduler.StartAsync()

	return func() {
		scheduler.Stop()
	}
}
