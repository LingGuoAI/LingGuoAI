package jobs

import (
	tasktypes "spiritFruit/pkg/asynq"

	hibikenAsynq "github.com/hibiken/asynq"
)

func NewServeMux() *hibikenAsynq.ServeMux {
	mux := hibikenAsynq.NewServeMux()
	mux.HandleFunc(tasktypes.TypeGenerateScript, HandleGenerateScript)
	mux.HandleFunc(tasktypes.TypeGenerateImage, HandleGenerateImage)
	mux.HandleFunc(tasktypes.TypeGenerateCharacters, HandleGenerateCharacters)
	mux.HandleFunc(tasktypes.TypeExtractScenes, HandleExtractScenes)
	mux.HandleFunc(tasktypes.TypeGenerateSceneImage, HandleGenerateSceneImage)
	mux.HandleFunc(tasktypes.TypeGenerateShots, HandleGenerateShots)
	mux.HandleFunc(tasktypes.TypeExtractProps, HandleExtractPropsTask)
	mux.HandleFunc(tasktypes.TypeGeneratePropImage, HandleGeneratePropImageTask)
	mux.HandleFunc(tasktypes.TypeExtractFramePrompt, HandleExtractFramePromptTask)
	mux.HandleFunc(tasktypes.TypeGenerateFrameImage, HandleGenerateFrameImageTask)
	mux.HandleFunc(tasktypes.TypeGenerateVideo, HandleGenerateVideoTask)
	mux.HandleFunc(tasktypes.TypeMergeVideo, HandleMergeVideoTask)
	mux.HandleFunc(tasktypes.TypeScriptWorkflowStage, HandleScriptWorkflow)

	return mux
}
