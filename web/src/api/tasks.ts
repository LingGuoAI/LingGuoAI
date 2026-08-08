import { request } from "@/utils/request";

// ==========================================
// 任务查询 (通用)
// ==========================================

// @Summary 查询任务详情 (轮询用)
// @Router /tasks/{id} [get]
export const findTasks = (id: number | string) => {
  return request.get({
    url: `/tasks/${id}`,
  });
};

// ==========================================
// AI 任务提交
// ==========================================

// 1. AI生成剧本
export const generateScriptTask = (data: {
  projectId: number | string;
  scriptId: number | string;
  prompt: string;
}) => {
  return request.post({
    url: "/scripts/generate",
    data,
  });
};

// 2. AI生成角色 (从剧本提取)
export const generateCharactersTask = (data: {
  projectId: number | string;
  count: number;
  outline?: string;
}) => {
  return request.post({
    url: "/tasks/generateCharacters",
    data,
  });
};

// 3. AI提取场景 (从剧本提取)
export const extractScenesTask = (data: { scriptId: number | string }) => {
  return request.post({
    url: "/tasks/extractScenes",
    data,
  });
};

// 4. AI智能拆分分镜
export const generateShotsTask = (data: {
  scriptId: number | string;
  model?: string;
}) => {
  return request.post({
    url: "/tasks/generateShots",
    data,
  });
};

// 5. 批量生成角色图片
export const batchGenerateCharacterImagesTask = (data: {
  characterIds: number[];
}) => {
  return request.post({
    url: "/tasks/batchGenerateCharacterImages",
    data,
  });
};

// 6. 单个生成角色图片
export const generateCharacterImageTask = (data: { characterId: number }) => {
  return request.post({
    url: "/tasks/generateCharacterImage",
    data,
  });
};

// 7. 单个生成场景图片
export const generateSceneImageTask = (data: { sceneId: number }) => {
  return request.post({
    url: "/tasks/generateSceneImage",
    data,
  });
};

// 8. 批量生成场景图片
export const batchGenerateSceneImagesTask = (data: { sceneIds: number[] }) => {
  return request.post({
    url: "/tasks/batchGenerateSceneImages",
    data,
  });
};

// 9. 从剧本提取道具
export const extractPropsTask = (data: { episodeId: number | string }) => {
  return request.post({
    url: "/tasks/extractProps",
    data,
  });
};

// 10. 单个生成道具图片
export const generatePropImageTask = (data: { propId: number | string }) => {
  return request.post({
    url: "/tasks/generatePropImage",
    data,
  });
};

// 11. 批量生成道具图片
export const batchGeneratePropImagesTask = (data: { propIds: number[] }) => {
  return request.post({
    url: "/tasks/batchGeneratePropImages",
    data,
  });
};

// 12. 提取分镜帧提示词
export const extractFramePromptTask = (data: {
  shotId: number | string;
  frameType: string;
  model?: string;
}) => {
  return request.post({
    url: "/tasks/extractPrompt",
    data,
  });
};

// 13. 根据帧提示词生成图片
export const generateImageByPromptTask = (data: {
  shotId: number | string;
  frameType: string;
  prompt: string;
  model?: string;
}) => {
  return request.post({
    url: "/tasks/generateImageByPrompt",
    data,
  });
};

// 14. 生成视频任务
export const generateVideoTask = (data: {
  projectId: number | string;
  shotId: number | string;
  model: string;
  duration: number;
  prompt: string;
  referenceMode: string;
  imageUrl?: string;
  firstFrameUrl?: string;
  lastFrameUrl?: string;
  imageUrls?: string[];
}) => {
  return request.post({
    url: "/tasks/generateVideo",
    data,
  });
};

// 15. 合并视频任务
export const mergeVideoTask = (data: {
  projectId: number | string;
  episodeNumber: number | string;
  scriptId?: number | string;
  clips: Array<{
    assetId?: number | string | null;
    shotId: string | number;
    order: number;
    startTime: number;
    endTime: number;
    duration: number;
    url?: string;
    transition?: Record<string, any>;
  }>;
}) => {
  return request.post({
    url: "/tasks/mergeVideo",
    data,
  });
};
