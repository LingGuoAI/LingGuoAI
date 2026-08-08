import { request } from '@/utils/request';

export const createScriptWorkflow = (data: any) => request.post({ url: '/script-workflows', data });
export const getScriptWorkflows = (params: any) => request.get({ url: '/script-workflows', params });
export const getScriptWorkflow = (id: number | string) => request.get({ url: `/script-workflows/${id}` });
export const runWorkflowStage = (id: number | string, data: any) => request.post({ url: `/script-workflows/${id}/run`, data });
export const generateWorkflowEpisodes = (id: number | string, data: any) => request.post({ url: `/script-workflows/${id}/episodes/generate`, data });
export const confirmWorkflowStep = (id: number | string, stepKey: string) => request.post({ url: `/script-workflows/${id}/confirm`, data: { stepKey } });
export const confirmWorkflowEpisode = (id: number | string, episodeNo: number) => request.post({ url: `/script-workflows/${id}/episodes/${episodeNo}/confirm` });
export const updateWorkflowEpisode = (id: number | string, episodeNo: number, contentText: string) => request.put({ url: `/script-workflows/${id}/episodes/${episodeNo}`, data: { contentText } });
export const getWorkflowEpisodeVersions = (id: number | string, episodeNo: number) => request.get({ url: `/script-workflows/${id}/episodes/${episodeNo}/versions` });
