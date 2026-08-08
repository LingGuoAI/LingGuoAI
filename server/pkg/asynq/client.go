package asynq

import (
	"encoding/json"
	"fmt"
	"github.com/hibiken/asynq"
	"spiritFruit/pkg/config"
	"spiritFruit/pkg/console"
	"sync"
	"time"
)

var (
	client *asynq.Client
	once   sync.Once
)

// GetClient 获取/初始化 Asynq Client 单例
func GetClient() *asynq.Client {
	once.Do(func() {
		redisOpt := asynq.RedisClientOpt{
			Addr:     fmt.Sprintf("%v:%v", config.GetString("redis.host"), config.GetString("redis.port")),
			Password: config.GetString("redis.password"),
			DB:       config.GetInt("redis.database_async"),
		}
		client = asynq.NewClient(redisOpt)
	})
	return client
}

// EnqueueGenerateScript 投递剧本生成任务
func EnqueueGenerateScript(payload GenerateScriptPayload) (*asynq.TaskInfo, error) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	task := asynq.NewTask(TypeGenerateScript, bytes)
	return GetClient().Enqueue(task, asynq.Queue("critical"))
}

func EnqueueScriptWorkflow(payload ScriptWorkflowPayload) (*asynq.TaskInfo, error) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return GetClient().Enqueue(asynq.NewTask(TypeScriptWorkflowStage, bytes), asynq.Queue("critical"), asynq.Timeout(10*time.Minute), asynq.MaxRetry(2))
}

// EnqueueGenerateImage 投递图片生成任务
func EnqueueGenerateImage(payload GenerateImagePayload) (*asynq.TaskInfo, error) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	task := asynq.NewTask(TypeGenerateImage, bytes)
	// 图片生成可能较慢，放入 default 队列
	info, err := GetClient().Enqueue(task, asynq.Queue("default"))
	if err != nil {
		console.Error(fmt.Sprintf("投递图片任务失败: %v", err))
		return nil, err
	}
	return info, nil
}

// EnqueueGenerateCharacters 投递角色生成任务
func EnqueueGenerateCharacters(payload GenerateCharactersPayload) (*asynq.TaskInfo, error) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	// 角色生成不算特别耗时，放入 default 队列
	task := asynq.NewTask(TypeGenerateCharacters, bytes)
	return GetClient().Enqueue(task, asynq.Queue("default"))
}

// EnqueueExtractScenes 投递场景提取任务
func EnqueueExtractScenes(payload ExtractScenesPayload) (*asynq.TaskInfo, error) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	// 场景提取涉及整集剧本分析，耗时较长，放入 default 队列
	task := asynq.NewTask(TypeExtractScenes, bytes)
	return GetClient().Enqueue(task, asynq.Queue("default"))
}

// EnqueueGenerateSceneImage 投递场景生图任务
func EnqueueGenerateSceneImage(payload GenerateSceneImagePayload) (*asynq.TaskInfo, error) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	// 放入 default 队列
	task := asynq.NewTask(TypeGenerateSceneImage, bytes)
	return GetClient().Enqueue(task, asynq.Queue("default"))
}

// EnqueueGenerateShots 投递分镜生成任务
func EnqueueGenerateShots(payload GenerateShotsPayload) (*asynq.TaskInfo, error) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	// 分镜生成耗时较长，建议放入 default 或 critical 队列
	task := asynq.NewTask(TypeGenerateShots, bytes)
	return GetClient().Enqueue(
		task,
		asynq.Queue("default"),
		asynq.Timeout(10*time.Minute), // 设置任务最大执行时间为 10 分钟
		asynq.MaxRetry(3),             // (可选) 如果任务失败，最多重试 3 次
	)
}

// EnqueueExtractProps 投递[从剧本提取道具]任务
func EnqueueExtractProps(payload ExtractPropsPayload) (*asynq.TaskInfo, error) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	// 提取任务属于文本分析，耗时中等，放入 default 队列
	task := asynq.NewTask(TypeExtractProps, bytes)
	return GetClient().Enqueue(task, asynq.Queue("default"))
}

// EnqueueGeneratePropImage 投递[道具生图]任务
func EnqueueGeneratePropImage(payload GeneratePropImagePayload) (*asynq.TaskInfo, error) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	// 生图任务耗时较长，放入 default 队列
	task := asynq.NewTask(TypeGeneratePropImage, bytes)
	return GetClient().Enqueue(task, asynq.Queue("default"))
}

// EnqueueExtractFramePrompt 投递提取帧提示词任务
func EnqueueExtractFramePrompt(payload ExtractFramePromptPayload) (*asynq.TaskInfo, error) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	// 文本提取任务一般较快，放入 default 队列
	task := asynq.NewTask(TypeExtractFramePrompt, bytes)
	info, err := GetClient().Enqueue(task, asynq.Queue("default"))
	if err != nil {
		console.Error(fmt.Sprintf("投递提取帧提示词任务失败: %v", err))
		return nil, err
	}
	return info, nil
}

// EnqueueGenerateFrameImage 投递根据帧提示词生成图片任务
func EnqueueGenerateFrameImage(payload GenerateFrameImagePayload) (*asynq.TaskInfo, error) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	// 生图任务耗时较长，放入 default 队列
	task := asynq.NewTask(TypeGenerateFrameImage, bytes)
	info, err := GetClient().Enqueue(task, asynq.Queue("default"))
	if err != nil {
		console.Error(fmt.Sprintf("投递分镜帧生图任务失败: %v", err))
		return nil, err
	}
	return info, nil
}

// EnqueueGenerateVideo 投递视频生成任务
func EnqueueGenerateVideo(payload GenerateVideoPayload) (*asynq.TaskInfo, error) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	// 视频生成非常耗时，放入 default 或 critical 队列
	task := asynq.NewTask(TypeGenerateVideo, bytes)
	info, err := GetClient().Enqueue(task, asynq.Queue("default"), asynq.Timeout(10*time.Minute))
	if err != nil {
		console.Error(fmt.Sprintf("投递生成视频任务失败: %v", err))
		return nil, err
	}
	return info, nil
}

// EnqueueMergeVideo 投递视频合并任务
func EnqueueMergeVideo(payload MergeVideoPayload) (*asynq.TaskInfo, error) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	// 合并视频属于 CPU 密集型极高的操作，建议放入 default 或 dedicated queue
	task := asynq.NewTask(TypeMergeVideo, bytes)
	info, err := GetClient().Enqueue(task, asynq.Queue("default"))
	if err != nil {
		console.Error(fmt.Sprintf("投递视频合成任务失败: %v", err))
		return nil, err
	}
	return info, nil
}
