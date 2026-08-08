<template>
    <div class="workflow-container">
        <div class="workflow-header">
            <div class="header-left">
                <t-button variant="text" shape="circle" @click="goBack">
                    <template #icon><t-icon name="arrow-left" /></template>
                </t-button>
                <div class="header-title">
                    <span class="title">{{ project?.title || '加载中...' }}</span>
                    <t-tag :theme="getStatusTheme(project?.status)" variant="light">{{ getStatusText(project?.status)
                    }}</t-tag>
                </div>
            </div>
            <div class="header-center">
                <t-steps :current="currentStep" readonly theme="dot" class="workflow-steps">
                    <t-step-item :title="`第 ${currentScriptNumber} 集剧本`" content="撰写剧本" />
                    <t-step-item title="角色定妆" content="提取并生成角色" />
                    <t-step-item title="分镜拆解" content="场景提取与分镜" />
                </t-steps>
            </div>
            <div class="header-right">
                <t-button theme="default" variant="outline" size="small" @click="initData">
                    <template #icon><t-icon name="refresh" /></template>
                </t-button>
            </div>
        </div>

        <div class="stage-area" v-loading="loading">
            <div v-show="currentStep === 0" class="stage-wrapper">
                <t-card :bordered="false" class="full-height-card">
                    <div v-if="!hasScriptContent && !showScriptInput" class="empty-state-wrapper">
                        <t-empty :description="`尚未创建第 ${currentScriptNumber} 集内容`">
                            <template #action>
                                <t-button theme="primary" size="large" @click="startCreateChapter">
                                    <template #icon><t-icon name="file-add" /></template> 开始创作
                                </t-button>
                            </template>
                        </t-empty>
                    </div>

                    <div v-if="showScriptInput" class="script-editor-container">
                        <div class="editor-toolbar">
                            <div class="toolbar-title">剧本编辑器 (第 {{ currentScriptNumber }} 集)</div>
                            <t-button theme="primary" variant="outline" @click="handleGenerateScriptAI"
                                :loading="generatingScript">
                                <template #icon><t-icon name="magic" /></template> AI 灵感生成
                            </t-button>
                        </div>
                        <t-textarea v-model="scriptContent" placeholder="请输入剧本内容..." class="script-textarea"
                            :autosize="{ minRows: 15 }" :disabled="generatingScript" />
                        <div class="editor-footer">
                            <t-button theme="default" style="margin-right: 12px" @click="cancelEdit">取消</t-button>
                            <t-button theme="primary" @click="handleSaveScript(false)" :loading="saving"
                                :disabled="!scriptContent.trim()">
                                <template #icon><t-icon name="check" /></template> 保存章节
                            </t-button>
                        </div>
                    </div>

                    <div v-if="hasScriptContent && !showScriptInput" class="script-preview-container">
                        <div class="preview-header">
                            <div class="ph-left">
                                <h3>第 {{ currentScriptNumber }} 集剧本</h3>
                                <t-tag theme="success" variant="light">已保存</t-tag>
                                <span class="update-time" v-if="currentScriptData.updatedAt">更新于: {{
                                    formatTime(currentScriptData.updatedAt) }}</span>
                            </div>
                            <t-button theme="primary" variant="text" @click="enterEditMode">
                                <template #icon><t-icon name="edit" /></template>修改剧本
                            </t-button>
                        </div>
                        <div class="preview-content">
                            <t-textarea :value="currentScriptData.content" readonly class="readonly-textarea"
                                :autosize="{ minRows: 10 }" />
                        </div>
                        <div class="step-actions">
                            <t-button theme="primary" size="large" @click="nextStep">
                                下一步：角色定妆 <template #suffix><t-icon name="chevron-right" /></template>
                            </t-button>
                        </div>
                    </div>
                </t-card>
            </div>

            <div v-show="currentStep === 1" class="stage-wrapper">
                <t-card :bordered="false" class="full-height-card">
                    <div class="toolbar-section">
                        <div class="toolbar-left">
                            <t-checkbox :checked="checkAll" :indeterminate="isIndeterminate" @change="handleSelectAll"
                                :disabled="characterList.length === 0">全选 ({{ selectedCharacterIds.length }}/{{
                                    characterList.length }})</t-checkbox>
                        </div>
                        <div class="toolbar-right">
                            <t-button theme="default" variant="outline" :loading="parsingCharacters"
                                @click="parseScriptToCharacters"><template #icon><t-icon
                                        name="user-search" /></template>从剧本智能提取</t-button>
                            <t-button theme="primary" variant="outline" :disabled="selectedCharacterIds.length === 0"
                                :loading="batchGenerating" @click="batchGenerateCharacterImages"><template #icon><t-icon
                                        name="image" /></template>批量生成选中形象</t-button>
                            <t-button theme="success" @click="nextStep" :disabled="!allCharactersHaveImages">下一步：分镜拆解
                                <template #suffix><t-icon name="chevron-right" /></template></t-button>
                        </div>
                    </div>
                    <div class="character-grid">
                        <div class="char-card add-card" @click="openAddCharacterDialog">
                            <div class="add-content"><t-icon name="add" size="32px" /><span>手动添加角色</span></div>
                        </div>
                        <div v-for="char in characterList" :key="char.id" class="char-card"
                            :class="{ 'is-selected': selectedCharacterIds.includes(char.id) }">
                            <div class="card-select">
                                <t-checkbox :checked="selectedCharacterIds.includes(char.id)"
                                    @change="(val) => toggleSelection(val, char.id)" />
                            </div>
                            <div class="char-image">
                                <t-image v-if="char.avatarUrl" :src="getImageUrl(char.avatarUrl)" fit="cover"
                                    class="img-box" />
                                <div v-else class="img-placeholder"><t-avatar size="large">{{ char.name ? char.name[0] :
                                    '?'
                                        }}</t-avatar></div>
                                <div v-if="generatingCharacterIds.includes(char.id)" class="loading-mask"><t-loading
                                        text="AI生成中..." size="small"></t-loading></div>
                            </div>
                            <div class="char-info">
                                <div class="info-head"><span class="name">{{ char.name }}</span><t-tag size="small"
                                        :theme="getRoleTheme(char.roleType)">{{ getRoleText(char.roleType) }}</t-tag>
                                </div>
                                <div class="desc text-ellipsis-2" :title="char.visualPrompt || char.appearanceDesc">{{
                                    char.visualPrompt || char.appearanceDesc || '暂无描述' }}</div>
                                <t-link theme="primary" size="small"
                                    @click="openEditCharacterDialog(char)">编辑详情</t-link>
                            </div>
                            <div class="char-actions">
                                <t-tooltip content="AI生成形象"><t-button shape="circle" size="small" theme="primary"
                                        :disabled="generatingCharacterIds.includes(char.id)"
                                        @click="generateCharacterImage(char)"><t-icon
                                            name="magic" /></t-button></t-tooltip>
                                <t-tooltip content="编辑详情/上传图片"><t-button shape="circle" size="small" variant="outline"
                                        @click="openEditCharacterDialog(char)"><t-icon
                                            name="edit" /></t-button></t-tooltip>
                                <t-popconfirm content="确认删除?" @confirm="deleteCharacter(char)"><t-button shape="circle"
                                        size="small" theme="danger" variant="text"><t-icon
                                            name="delete" /></t-button></t-popconfirm>
                            </div>
                        </div>
                    </div>
                </t-card>
            </div>

            <div v-show="currentStep === 2" class="stage-wrapper">
                <t-card :bordered="false" class="full-height-card">
                    <template #header>
                        <div class="card-header-flex">
                            <div class="header-info">

                                <t-icon name="film" size="24px" style="color: var(--td-brand-color)" />
                                <span class="title">场景与分镜</span>
                            </div>
                            <div class="header-actions">
                                <div v-if="isShotTaskRunning"
                                    style="width: 120px; margin-right: 10px; display: flex; align-items: center;">
                                    <t-progress theme="line" :percentage="shotProcessPercent" :label="false" />
                                    <span style="font-size: 12px; margin-left: 8px; color: var(--td-brand-color);">{{
                                        shotProcessPercent }}%</span>
                                </div>

                                <t-button theme="warning" variant="outline" @click="handleExtractScenes"
                                    :loading="extractingScenes" :disabled="isShotTaskRunning">
                                    <template #icon><t-icon name="image" /></template> 提取场景
                                </t-button>

                                <t-button theme="default" @click="regenerateShots"
                                    :disabled="!shotList?.length || isShotTaskRunning">
                                    <template #icon><t-icon name="refresh" /></template> 重新拆分
                                </t-button>
                            </div>
                        </div>
                    </template>

                    <div class="split-view">
                        <div class="scenes-panel" v-if="sceneList.length > 0">
                            <div class="panel-header">
                                <span>场景列表 ({{ sceneList.length }})</span>
                                <t-button size="small" variant="text" theme="primary" @click="openAddSceneDialog">
                                    <template #icon><t-icon name="add" /></template>新增
                                </t-button>
                            </div>
                            <div class="scene-batch-bar">
                                <t-checkbox :checked="checkAllScenes" :indeterminate="isSceneIndeterminate"
                                    @change="handleSelectAllScenes">全选</t-checkbox>
                                <t-tooltip content="批量生成选中场景图片">
                                    <t-button shape="square" variant="outline" theme="primary" size="small"
                                        :disabled="selectedSceneIds.length === 0" @click="batchGenerateSceneImages"
                                        :loading="batchGeneratingScenes">
                                        <template #icon><t-icon name="magic" /></template>
                                    </t-button>
                                </t-tooltip>
                            </div>
                            <div class="scene-list">
                                <div v-for="scene in sceneList" :key="scene.id" class="scene-item"
                                    :class="{ 'is-selected': selectedSceneIds.includes(scene.id) }">
                                    <div class="scene-image-area">
                                        <div class="scene-select-box">
                                            <t-checkbox :checked="selectedSceneIds.includes(scene.id)"
                                                @change="(val) => toggleSceneSelection(val, scene.id)" />
                                        </div>
                                        <t-image v-if="scene.imageUrl" :src="getImageUrl(scene.imageUrl)" fit="cover"
                                            class="s-img" @click="openEditSceneDialog(scene)" />
                                        <div v-else class="img-placeholder" @click="openEditSceneDialog(scene)"><t-icon
                                                name="image" size="24px" /><span class="placeholder-text">点击上传</span>
                                        </div>
                                        <div v-if="generatingSceneIds.includes(scene.id)" class="loading-mask">
                                            <t-loading size="small" text="AI生成中..." />
                                        </div>
                                    </div>
                                    <div class="scene-content">
                                        <div class="scene-header-row"><span class="scene-name" :title="scene.name">{{
                                            scene.name
                                                }}</span><t-tag size="small" variant="light" theme="warning">{{
                                                    scene.time }}</t-tag></div>
                                        <div class="scene-loc"><t-icon name="location" /> {{ scene.location }}</div>
                                        <div class="scene-desc text-ellipsis-2" :title="scene.atmosphere">{{
                                            scene.atmosphere || '暂无描述' }}
                                        </div>
                                    </div>
                                    <div class="scene-footer">
                                        <t-row :gutter="0" style="width: 100%; text-align: center;">
                                            <t-col :span="4">
                                                <t-tooltip content="AI生成图片">
                                                    <t-button variant="text" theme="primary" size="small" block
                                                        @click="generateSceneImage(scene)"
                                                        :disabled="generatingSceneIds.includes(scene.id)">
                                                        <template #icon><t-icon name="image" /></template>
                                                    </t-button>
                                                </t-tooltip>
                                            </t-col>
                                            <t-col :span="4"><t-tooltip content="编辑详情/上传图"><t-button variant="text"
                                                        size="small" block @click="openEditSceneDialog(scene)"><t-icon
                                                            name="edit" /></t-button></t-tooltip></t-col>
                                            <t-col :span="4"><t-popconfirm content="确定删除该场景吗?"
                                                    @confirm="deleteScene(scene)"><t-button variant="text"
                                                        theme="danger" size="small" block><t-icon
                                                            name="delete" /></t-button></t-popconfirm></t-col>
                                        </t-row>
                                    </div>
                                </div>
                            </div>
                        </div>

                        <div class="shots-panel" :class="{ 'full-width': sceneList.length === 0 }">
                            <div class="panel-header" v-if="sceneList.length > 0">分镜列表 ({{ shotList.length }})</div>

                            <div v-if="shotList.length > 0">
                                <t-table :data="shotList" :columns="shotColumns" row-key="id" stripe hover
                                    :max-height="600" table-layout="auto">
                                    <template #shotInfo="{ row }">
                                        <t-space size="small" direction="vertical" style="gap: 2px">
                                            <t-tag size="small" variant="light" theme="primary" v-if="row.shotType">{{
                                                row.shotType }}</t-tag>
                                            <t-tag size="small" variant="outline" v-if="row.angle">{{ row.angle
                                            }}</t-tag>
                                            <t-tag size="small" variant="outline" v-if="row.cameraMovement">{{
                                                row.cameraMovement }}</t-tag>
                                        </t-space>
                                    </template>
                                    <template #sceneInfo="{ row }">
                                        <div style="font-size: 12px; line-height: 1.4;">
                                            <div style="font-weight: bold; margin-bottom: 2px;"><t-icon name="location"
                                                    style="color: var(--td-brand-color);" /> {{
                                                        getSceneName(row.sceneId) || row.location || '未关联场景' }}</div>
                                            <div v-if="row.time" style="color: var(--td-text-color-secondary);">{{
                                                row.time }}</div>
                                        </div>
                                    </template>
                                    <template #visual="{ row }">
                                        <div style="font-size: 12px; display: flex; flex-direction: column; gap: 4px;">
                                            <div v-if="row.action" style="color: var(--td-text-color-primary);"
                                                class="text-ellipsis-2" :title="row.action"><t-tag size="small"
                                                    theme="danger" variant="light">动作</t-tag> {{ row.action }}</div>
                                            <div style="color: var(--td-text-color-secondary);" class="text-ellipsis-2"
                                                :title="row.visualDesc"><t-tag size="small" variant="light">画面</t-tag>
                                                {{ row.visualDesc || '暂无画面描述' }}</div>
                                        </div>
                                    </template>
                                    <template #duration="{ row }">{{ (row.durationMs || 3000) / 1000 }}s</template>
                                    <template #operation="{ row }">
                                        <t-space size="small">
                                            <t-button shape="circle" variant="text" theme="primary"
                                                @click="openEditShotDialog(row)"><template #icon><t-icon
                                                        name="edit" /></template></t-button>
                                            <t-popconfirm content="删除此分镜?" @confirm="handleDeleteShotFromList(row.id)">
                                                <t-button shape="circle" variant="text" theme="danger"><template
                                                        #icon><t-icon name="delete" /></template></t-button>
                                            </t-popconfirm>
                                        </t-space>
                                    </template>
                                </t-table>

                                <div class="step-actions mt-4" v-if="shotList.length > 0">
                                    <t-button theme="default" @click="prevStep" :disabled="isShotTaskRunning">
                                        上一步
                                    </t-button>

                                    <t-button theme="success" @click="goToScriptEditor" size="large"
                                        :disabled="isShotTaskRunning">
                                        {{ isShotTaskRunning ? `AI 拆解中 (${shotProcessPercent}%)...` : '进入专业制作' }}

                                        <template #icon v-if="!isShotTaskRunning">
                                            <t-icon name="arrow-right" />
                                        </template>
                                    </t-button>
                                </div>
                            </div>

                            <div v-else class="empty-state-wrapper">
                                <t-empty description="暂无分镜数据">
                                    <template #action>
                                        <t-space>
                                            <t-button theme="warning" variant="outline" size="large"
                                                @click="handleExtractScenes" :loading="extractingScenes"><t-icon
                                                    name="image" /> 第一步：提取场景</t-button>
                                            <t-button theme="primary" size="large" @click="generateShots"
                                                :loading="generatingShots"><t-icon name="magic" /> 第二步：智能拆分</t-button>
                                        </t-space>
                                    </template>
                                </t-empty>
                            </div>
                        </div>
                    </div>
                </t-card>
            </div>
        </div>

        <t-dialog v-model:visible="addCharacterDialogVisible" :header="isEditMode ? '编辑角色' : '添加新角色'" width="600px"
            :confirm-btn="{ content: '保存', theme: 'primary', loading: saving }" @confirm="handleCharacterSubmit">
            <t-form :data="newCharacter" label-align="top">
                <t-form-item label="角色形象" name="avatarUrl">
                    <t-upload v-model="characterFileList" :action="uploadConfig.action" :headers="uploadConfig.headers"
                        :show-file-list="false" accept="image/*"
                        @success="(ctx) => handleUploadSuccess(ctx, false, 'character')">
                        <div class="upload-trigger" v-if="!newCharacter.avatarUrl"
                            style="width:120px;height:120px;border:1px dashed #ccc;display:flex;align-items:center;justify-content:center;cursor:pointer">
                            <t-icon name="add" size="24px" />
                        </div>
                        <t-image v-else :src="getImageUrl(newCharacter.avatarUrl)" fit="cover"
                            style="width:120px;height:120px" />
                    </t-upload>
                </t-form-item>
                <t-form-item label="角色名称" name="name" required><t-input v-model="newCharacter.name" /></t-form-item>
                <t-form-item label="类型" name="roleType"><t-select v-model="newCharacter.roleType"
                        :options="roleOptions" /></t-form-item>
                <t-form-item label="外貌描述" name="appearanceDesc"><t-textarea
                        v-model="newCharacter.appearanceDesc" /></t-form-item>
            </t-form>
        </t-dialog>

        <t-dialog v-model:visible="addSceneDialogVisible" :header="isSceneEditMode ? '编辑场景' : '添加新场景'" width="500px"
            :confirm-btn="{ content: '保存', theme: 'primary', loading: saving }" @confirm="handleSceneSubmit">
            <t-form :data="newScene" label-align="top">
                <t-form-item label="场景参考图" name="imageUrl">
                    <t-upload v-model="sceneFileList" :action="uploadConfig.action" :headers="uploadConfig.headers"
                        :show-file-list="false" accept="image/*"
                        @success="(ctx) => handleUploadSuccess(ctx, false, 'scene')">
                        <div class="upload-trigger" v-if="!newScene.imageUrl"
                            style="width:100%;height:160px;border:1px dashed #ccc;display:flex;align-items:center;justify-content:center;cursor:pointer">
                            <t-icon name="add" size="24px" />
                        </div>
                        <t-image v-else :src="getImageUrl(newScene.imageUrl)" fit="cover"
                            style="width:100%;height:160px" />
                    </t-upload>
                </t-form-item>
                <t-form-item label="场景名称" name="name" required><t-input v-model="newScene.name" /></t-form-item>
                <t-row :gutter="16">
                    <t-col :span="6"><t-form-item label="地点" name="location"><t-input
                                v-model="newScene.location" /></t-form-item></t-col>
                    <t-col :span="6"><t-form-item label="时间" name="time"><t-select v-model="newScene.time"
                                :options="['白天', '夜晚', '黄昏', '清晨'].map(v => ({ label: v, value: v }))" /></t-form-item></t-col>
                </t-row>
                <t-form-item label="氛围/描述" name="atmosphere"><t-textarea v-model="newScene.atmosphere" /></t-form-item>
            </t-form>
        </t-dialog>

        <t-dialog v-model:visible="shotDialog.visible" header="编辑分镜详情" width="800px"
            :confirm-btn="{ content: '保存修改', theme: 'primary', loading: shotDialog.loading }"
            @confirm="handleShotSubmit">
            <t-form :data="shotFormData" label-align="left" label-width="100px"
                style="max-height: 60vh; overflow-y: auto; padding-right: 16px;">
                <div style="font-weight: bold; margin-bottom: 12px; color: var(--td-brand-color);">基础设置</div>
                <t-row :gutter="16">
                    <t-col :span="6"><t-form-item label="镜头标题" name="title"><t-input
                                v-model="shotFormData.title" /></t-form-item></t-col>
                    <t-col :span="6"><t-form-item label="镜头序号" name="sequenceNo"><t-input-number
                                v-model="shotFormData.sequenceNo" :min="1" /></t-form-item></t-col>
                </t-row>
                <t-row :gutter="16">
                    <t-col :span="6"><t-form-item label="所属场景" name="sceneId"><t-select v-model="shotFormData.sceneId"
                                :options="sceneOptions" filterable /></t-form-item></t-col>
                    <t-col :span="6"><t-form-item label="时长 (秒)" name="durationSec"><t-input-number
                                v-model="shotFormData.durationSec" :min="1" :max="60" /></t-form-item></t-col>
                </t-row>
                <t-row :gutter="16">
                    <t-col :span="8"><t-form-item label="地点" name="location"><t-input
                                v-model="shotFormData.location" /></t-form-item></t-col>
                    <t-col :span="4"><t-form-item label="时间" name="time"><t-input
                                v-model="shotFormData.time" /></t-form-item></t-col>
                </t-row>
                <t-divider style="margin: 16px 0;" />
                <div style="font-weight: bold; margin-bottom: 12px; color: var(--td-brand-color);">视效运镜</div>
                <t-row :gutter="16">
                    <t-col :span="4"><t-form-item label="景别" name="shotType"><t-select v-model="shotFormData.shotType"
                                :options="['大远景', '远景', '全景', '中景', '近景', '特写', '大特写'].map(v => ({ label: v, value: v }))" /></t-form-item></t-col>
                    <t-col :span="4"><t-form-item label="角度" name="angle"><t-select v-model="shotFormData.angle"
                                :options="['平视', '俯视', '仰视', '侧面', '背面', '鸟瞰'].map(v => ({ label: v, value: v }))" /></t-form-item></t-col>
                    <t-col :span="4"><t-form-item label="运镜" name="cameraMovement"><t-select
                                v-model="shotFormData.cameraMovement"
                                :options="['固定镜头', '推镜', '拉镜', '摇镜', '移镜', '跟镜', '环绕'].map(v => ({ label: v, value: v }))" /></t-form-item></t-col>
                </t-row>
                <t-divider style="margin: 16px 0;" />
                <div style="font-weight: bold; margin-bottom: 12px; color: var(--td-brand-color);">叙事与提示词</div>
                <t-form-item label="人物动作" name="action"><t-textarea v-model="shotFormData.action"
                        :autosize="{ minRows: 2 }" /></t-form-item>
                <t-form-item label="画面描述" name="visualDesc"><t-textarea v-model="shotFormData.visualDesc"
                        :autosize="{ minRows: 2 }" /></t-form-item>
                <t-form-item label="台词对话" name="dialogue"><t-textarea v-model="shotFormData.dialogue"
                        :autosize="{ minRows: 2 }" /></t-form-item>
                <t-form-item label="环境氛围" name="atmosphere"><t-textarea v-model="shotFormData.atmosphere"
                        :autosize="{ minRows: 2 }" /></t-form-item>
                <t-form-item label="生图提示词" name="imagePrompt"><t-textarea v-model="shotFormData.imagePrompt"
                        :autosize="{ minRows: 2 }" /></t-form-item>
                <t-form-item label="视频提示词" name="videoPrompt"><t-textarea v-model="shotFormData.videoPrompt"
                        :autosize="{ minRows: 2 }" /></t-form-item>
                <t-form-item label="音频提示词" name="audioPrompt"><t-textarea v-model="shotFormData.audioPrompt"
                        :autosize="{ minRows: 2 }" /></t-form-item>
            </t-form>
        </t-dialog>

    </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { MessagePlugin, DialogPlugin } from 'tdesign-vue-next'
import dayjs from 'dayjs'

// API
import { findProjects } from '@/api/projects'
import { createScripts, updateScripts, getScriptsList, findScripts } from '@/api/scripts'
import { getCharactersList, createCharacters, updateCharacters, deleteCharacters } from '@/api/characters'
import { getScenesList, createScenes, updateScenes, deleteScenes } from '@/api/scenes'
import { updateShots, getShotsList, createShots, deleteShots } from '@/api/shots'
import { getAssetsList } from '@/api/assets'
import { generateShotsTask, findTasks, generateScriptTask, extractScenesTask, generateCharactersTask, batchGenerateCharacterImagesTask, batchGenerateSceneImagesTask, generateSceneImageTask } from '@/api/tasks'
import { getImageUrl } from '@/utils/format'

const route = useRoute()
const router = useRouter()

// === 状态 ===
const loading = ref(false)
const currentStep = ref(0) // 0: 剧本, 1: 角色, 2: 分镜拆解, 3: 专业编辑
const project = ref<any>({})
const currentScriptData = ref<any>({})
const currentScriptNumber = ref(Number(route.params.episodeNumber) || 1)
const characterList = ref<any[]>([])
const sceneList = ref<any[]>([])
const shotList = ref<any[]>([])

// 编辑器 & 任务状态
const showScriptInput = ref(false)
const scriptContent = ref("")
const generatingScript = ref(false)
const saving = ref(false)
const uploading = ref(false)
const generatingShots = ref(false)
const parsingCharacters = ref(false)
const extractingScenes = ref(false)
const batchGenerating = ref(false)
const batchGeneratingScenes = ref(false)

// 选择状态
const selectedCharacterIds = ref<number[]>([])
const generatingCharacterIds = ref<number[]>([])
const selectedSceneIds = ref<number[]>([])
const generatingSceneIds = ref<number[]>([])

// 弹窗状态
const addCharacterDialogVisible = ref(false)
const isEditMode = ref(false)
const newCharacter = ref({ id: undefined, name: '', roleType: 'supporting', appearanceDesc: '', personality: '', gender: '', avatarUrl: '' })
const roleOptions = [{ label: '主角', value: 'main' }, { label: '配角', value: 'supporting' }]
const characterFileList = ref<any[]>([])

const addSceneDialogVisible = ref(false)
const isSceneEditMode = ref(false)
const newScene = ref({ id: undefined, name: '', location: '', time: '白天', atmosphere: '', imageUrl: '' })
const sceneFileList = ref<any[]>([])

const shotDialog = reactive({ visible: false, loading: false })
const shotFormData = ref<any>({})

const getAuthToken = () => localStorage.getItem('token')
const uploadConfig = reactive({
    action: import.meta.env.VITE_API_URL + '/admin/v1/upload/singleUpload',
    headers: computed(() => ({ 'Authorization': `${getAuthToken()}` })),
    sizeLimit: 5 * 1024 * 1024,
})

// ========== 2. 计算属性 ==========
const hasScriptContent = computed(() => !!(currentScriptData.value?.id && currentScriptData.value?.content && currentScriptData.value?.content.trim().length > 0))
const hasCharacters = computed(() => characterList.value.length > 0)
const checkAll = computed(() => characterList.value.length > 0 && selectedCharacterIds.value.length === characterList.value.length)
const isIndeterminate = computed(() => selectedCharacterIds.value.length > 0 && selectedCharacterIds.value.length < characterList.value.length)
const allCharactersHaveImages = computed(() => characterList.value.length > 0 && characterList.value.every((c: any) => !!(c.avatarUrl || c.image_url)))
const checkAllScenes = computed(() => sceneList.value.length > 0 && selectedSceneIds.value.length === sceneList.value.length)
const isSceneIndeterminate = computed(() => selectedSceneIds.value.length > 0 && selectedSceneIds.value.length < sceneList.value.length)

// 记录分镜拆解的实时进度
const shotProcessPercent = ref(100) // 默认 100，表示没有任务在运行
// 是否正在进行分镜相关的 AI 任务
const isShotTaskRunning = computed(() => shotProcessPercent.value < 100 || generatingShots.value)

// 场景下拉列表
const sceneOptions = computed(() => sceneList.value.map(s => ({ label: s.name, value: s.id })))
// 列表展示中获取场景名
const getSceneName = (id: number) => {
    const s = sceneList.value.find(item => item.id === id)
    return s ? s.name : ''
}

// ========== 3. 工具函数 ==========
const pollTask = async (
    taskId: string,
    onSuccess: () => void | Promise<void>,
    onFail: () => void | Promise<void>,
    onProgress?: (percent: number) => void
) => {
    const timer = setInterval(async () => {
        try {
            const res = await findTasks(taskId)
            const payload = res.data?.data || res.data
            // /tasks/:id 返回 { task, events }，兼容旧接口直接返回 task 的格式。
            const data = payload?.task || payload
            const status = data?.status
            const statusName = data?.statusName || data?.status_name
            const percent = data?.process || 0 // 🔴 获取后端返回的 process 字段

            // 执行进度回调
            if (onProgress) onProgress(percent)

            if (status === 'completed' || status === 'succeeded' || statusName === 'succeeded' || status === 2 || percent >= 100) {
                clearInterval(timer)
                if (onProgress) onProgress(100)
                await onSuccess()
            } else if (status === 'failed' || statusName === 'failed' || status === 3 || status === 'cancelled' || statusName === 'cancelled' || status === 4) {
                clearInterval(timer)
                MessagePlugin.error(data?.errorMsg || data?.error_msg || data?.error || '任务执行失败')
                onFail()
            }
        } catch (e) {
            clearInterval(timer)
            onFail()
        }
    }, 2000)
}
const formatTime = (val: string) => dayjs(val).format('YYYY-MM-DD HH:mm')
const getStatusTheme = (s: any) => s === 2 ? 'success' : 'primary'
const getStatusText = (s: any) => s === 2 ? '已完成' : '制作中'
const getRoleTheme = (role: string) => ({ main: 'danger', supporting: 'primary', minor: 'default' }[role] || 'default')
const getRoleText = (role: string) => ({ main: '主角', supporting: '配角', minor: '路人' }[role] || '未知')

// ========== 4. 初始化 & 导航 ==========
const initData = async () => {
    loading.value = true
    try {
        await Promise.all([loadProjectInfo(), loadCharacters(), loadScenes()])
        await loadScriptDetail()
        if (currentScriptData.value?.id) await loadShots(currentScriptData.value.id)
    } catch (e) { console.error(e) } finally { loading.value = false }
}

const goToScriptEditor = () => {
    router.push({
        name: 'ScriptEditor',
        params: { dramaId: project.value.id, episodeNumber: currentScriptNumber.value }
    })
}

// ========== 5. 数据加载 ==========
const loadProjectInfo = async () => { const res = await findProjects(route.params.id as string); if (res.code === 0) project.value = res.data }
const loadCharacters = async () => { const res = await getCharactersList({ projectId: route.params.id as string, pageSize: 100 }); if (res.code === 0) characterList.value = Array.isArray(res.data) ? res.data : (res.data?.list || []) }
const loadScenes = async () => { const res = await getScenesList({ projectId: route.params.id as string, pageSize: 100 }); if (res.code === 0) sceneList.value = Array.isArray(res.data) ? res.data : (res.data?.list || []) }
const loadScriptDetail = async () => {
    const listRes = await getScriptsList({ projectId: route.params.id, page: 1, pageSize: 100 })
    if (listRes.code === 0) {
        const list = Array.isArray(listRes.data) ? listRes.data : (listRes.data?.list || [])
        const targetScript = list.find((s: any) => Number(s.episodeNo) === currentScriptNumber.value)
        if (targetScript) {
            const detailRes = await findScripts(targetScript.id)
            if (detailRes.code === 0) {
                currentScriptData.value = detailRes.data
                if (detailRes.data.content) showScriptInput.value = false
            }
        } else { currentScriptData.value = {}; showScriptInput.value = false }
    }
}
const loadShots = async (scriptId: number) => {
    const res = await getShotsList({ scriptId: scriptId, pageSize: 1000 })
    if (res.code === 0) shotList.value = res.data?.list || res.data || []
}

// ========== 6. 剧本生成 ==========
const startCreateChapter = () => { scriptContent.value = ''; showScriptInput.value = true }
const enterEditMode = () => { scriptContent.value = currentScriptData.value.content || ''; showScriptInput.value = true }
const cancelEdit = () => { showScriptInput.value = false; if (!currentScriptData.value.id) scriptContent.value = '' }
const handleGenerateScriptAI = async () => {
    if (!project.value?.title) return MessagePlugin.warning('项目信息缺失')
    if (!currentScriptData.value.id) { try { await handleSaveScript(true) } catch { return } }
    generatingScript.value = true
    try {
        const res = await generateScriptTask({ projectId: project.value.id, scriptId: currentScriptData.value.id, prompt: `基于项目《${project.value.title}》生成第${currentScriptNumber.value}集剧本` })
        const taskId = res.data?.data?.task_id || res.data?.taskId || res.data?.task_id
        if ((res.code === 0 || res.status === 200 || res.status === 0) && taskId) {
            MessagePlugin.loading('AI 正在创作剧本...')
            pollTask(taskId, () => { generatingScript.value = false; MessagePlugin.success('AI 创作完成'); loadScriptDetail() }, () => generatingScript.value = false)
        } else { MessagePlugin.error('生成失败'); generatingScript.value = false }
    } catch { generatingScript.value = false; MessagePlugin.error('请求失败') }
}
const handleSaveScript = async (silent = false) => {
    if (!scriptContent.value.trim() && !silent) return MessagePlugin.warning('内容不能为空')
    if (!silent) saving.value = true
    try {
        const payload = { projectId: project.value.id, episodeNo: currentScriptNumber.value, title: `第${currentScriptNumber.value}集`, content: scriptContent.value || ' ', isLocked: 0 }
        if (currentScriptData.value?.id) { await updateScripts(currentScriptData.value.id, payload); if (!silent) MessagePlugin.success('更新成功') }
        else { await createScripts(payload); if (!silent) MessagePlugin.success('创建成功') }
        await loadScriptDetail(); if (!silent) showScriptInput.value = false
    } catch (e) { if (!silent) MessagePlugin.error('操作失败'); throw e }
    finally { if (!silent) saving.value = false }
}

// ========== 7. 角色 & 场景 ==========

const handleSelectAll = (checked: boolean) => {
    if (characterList.value.length === 0) return;
    if (checked) selectedCharacterIds.value = characterList.value.map((c: any) => c.id);
    else selectedCharacterIds.value = []
}
const toggleSelection = (checked: boolean, id: number) => {
    if (checked) {
        if (!selectedCharacterIds.value.includes(id)) selectedCharacterIds.value.push(id)
    } else {
        const idx = selectedCharacterIds.value.indexOf(id);
        if (idx > -1) selectedCharacterIds.value.splice(idx, 1);
    }
}
const handleSelectAllScenes = (checked: boolean) => {
    if (sceneList.value.length === 0) return;
    if (checked) selectedSceneIds.value = sceneList.value.map((c: any) => c.id);
    else selectedSceneIds.value = []
}
const toggleSceneSelection = (checked: boolean, id: number) => {
    if (checked) {
        if (!selectedSceneIds.value.includes(id)) selectedSceneIds.value.push(id)
    } else {
        const idx = selectedSceneIds.value.indexOf(id);
        if (idx > -1) selectedSceneIds.value.splice(idx, 1);
    }
}


const parseScriptToCharacters = async () => {
    if (!project.value?.id) return; parsingCharacters.value = true
    // 🔴 1. 检查当前是否已经有剧本内容
    const scriptText = currentScriptData.value?.content;
    if (!scriptText || scriptText.trim() === '') {
        MessagePlugin.warning('请先在第一步撰写并保存剧本'); parsingCharacters.value = false
        return;
    }
    try {
        const res = await generateCharactersTask({ projectId: parseInt(project.value.id), count: 5, outline: scriptText })
        const taskId = res.data?.data?.task_id || res.data?.taskId || res.data?.task_id
        if (taskId) {
            MessagePlugin.loading('AI 正在提取角色...')
            pollTask(taskId, async () => { parsingCharacters.value = false; await loadCharacters(); MessagePlugin.success('角色提取完成') }, () => parsingCharacters.value = false)
        } else { MessagePlugin.error('提交失败'); parsingCharacters.value = false }
    } catch { parsingCharacters.value = false }
}
const runBatchGeneration = async (type: 'character' | 'scene', ids: number[]) => {
    try {
        let res
        if (type === 'character') res = await batchGenerateCharacterImagesTask({ characterIds: ids })
        else res = await batchGenerateSceneImagesTask({ sceneIds: ids })
        const taskList = res.data?.data || res.data || []
        if (taskList.length > 0) {
            MessagePlugin.success(`已提交 ${taskList.length} 个任务`)
            if (type === 'character') selectedCharacterIds.value = []; else selectedSceneIds.value = []
            taskList.forEach((item: any) => {
                const id = type === 'character' ? item.character_id : item.scene_id; const taskId = item.task_id
                const trackingList = type === 'character' ? generatingCharacterIds : generatingSceneIds; const refreshFunc = type === 'character' ? loadCharacters : loadScenes
                if (!trackingList.value.includes(id)) trackingList.value.push(id)
                pollTask(taskId, () => { const idx = trackingList.value.indexOf(id); if (idx > -1) trackingList.value.splice(idx, 1); refreshFunc() }, () => { const idx = trackingList.value.indexOf(id); if (idx > -1) trackingList.value.splice(idx, 1) })
            })
        } else { MessagePlugin.warning('未创建任务') }
    } catch { MessagePlugin.error('任务提交失败') }
}
const generateCharacterImage = async (char: any) => { if (generatingCharacterIds.value.includes(char.id)) return; generatingCharacterIds.value.push(char.id); await runBatchGeneration('character', [char.id]) }
const batchGenerateCharacterImages = async () => { if (selectedCharacterIds.value.length === 0) return; batchGenerating.value = true; try { await runBatchGeneration('character', [...selectedCharacterIds.value]) } finally { batchGenerating.value = false } }


const generateSceneImage = async (scene: any) => {
    if (generatingSceneIds.value.includes(scene.id)) return; generatingSceneIds.value.push(scene.id)
    try {
        const res = await generateSceneImageTask({ sceneId: scene.id });
        const taskId = res.data?.task_id || res.data?.taskId
        if (taskId) {
            MessagePlugin.success('场景生图任务已提交')
            pollTask(taskId, () => { const idx = generatingSceneIds.value.indexOf(scene.id); if (idx > -1) generatingSceneIds.value.splice(idx, 1); loadScenes() }, () => { const idx = generatingSceneIds.value.indexOf(scene.id); if (idx > -1) generatingSceneIds.value.splice(idx, 1) })
        } else { const idx = generatingSceneIds.value.indexOf(scene.id); if (idx > -1) generatingSceneIds.value.splice(idx, 1) }
    } catch { const idx = generatingSceneIds.value.indexOf(scene.id); if (idx > -1) generatingSceneIds.value.splice(idx, 1); MessagePlugin.error('提交失败') }
}
const batchGenerateSceneImages = async () => { if (selectedSceneIds.value.length === 0) return; batchGeneratingScenes.value = true; try { await runBatchGeneration('scene', [...selectedSceneIds.value]) } finally { batchGeneratingScenes.value = false } }


const openAddSceneDialog = () => { isSceneEditMode.value = false; newScene.value = { id: undefined, name: '', location: '', time: '白天', atmosphere: '', visualPrompt: '' }; sceneFileList.value = []; addSceneDialogVisible.value = true }
const openEditSceneDialog = (scene: any) => { isSceneEditMode.value = true; newScene.value = { ...scene }; if (scene.visualPrompt) sceneFileList.value = [{ url: getImageUrl(scene.visualPrompt), name: '场景图' }]; else sceneFileList.value = []; addSceneDialogVisible.value = true }
const handleSceneSubmit = async () => { saving.value = true; try { let res; const payload = { projectId: project.value.id, ...newScene.value }; if (isSceneEditMode.value) res = await updateScenes(newScene.value.id, payload); else res = await createScenes(payload); if (res.code === 0) { addSceneDialogVisible.value = false; MessagePlugin.success(isSceneEditMode.value ? '更新成功' : '添加成功'); loadScenes() } else { MessagePlugin.error(res.message || '操作失败') } } catch { MessagePlugin.error('网络请求失败') } finally { saving.value = false } }
const deleteScene = async (scene: any) => { try { const res = await deleteScenes(scene.id); if (res.code === 0) { MessagePlugin.success('删除成功'); loadScenes() } else { MessagePlugin.error('删除失败') } } catch { MessagePlugin.error('删除请求失败') } }

const openAddCharacterDialog = () => { isEditMode.value = false; newCharacter.value = { id: undefined, name: '', roleType: 'supporting', appearanceDesc: '', personality: '', gender: '', avatarUrl: '' }; characterFileList.value = []; addCharacterDialogVisible.value = true }
const openEditCharacterDialog = (char: any) => { isEditMode.value = true; newCharacter.value = { ...char }; if (char.avatarUrl) characterFileList.value = [{ url: getImageUrl(char.avatarUrl), name: '角色图' }]; else characterFileList.value = []; addCharacterDialogVisible.value = true }
const handleCharacterSubmit = async () => { saving.value = true; try { let res; const payload = { projectId: project.value.id, ...newCharacter.value }; if (isEditMode.value) res = await updateCharacters(newCharacter.value.id, payload); else res = await createCharacters(payload); if (res.code === 0) { addCharacterDialogVisible.value = false; MessagePlugin.success(isEditMode.value ? '更新成功' : '添加成功'); loadCharacters() } else { MessagePlugin.error(res.message || '操作失败') } } catch { MessagePlugin.error('网络请求失败') } finally { saving.value = false } }
const deleteCharacter = async (char: any) => { try { const res = await deleteCharacters(char.id); if (res.code === 0) { MessagePlugin.success('删除成功'); loadCharacters() } else { MessagePlugin.error('删除失败') } } catch { MessagePlugin.error('删除请求失败') } }

const handleUploadSuccess = (ctx: any, isReupload: boolean, type: 'character' | 'scene') => {
    uploading.value = false; const response = ctx.response; if (response?.code === 0 || response?.code === 200) { const responseData = response.data; let fileUrl = responseData.file_url || responseData.url; if (fileUrl && fileUrl.startsWith('/')) fileUrl = import.meta.env.VITE_API_URL.replace(/\/admin\/v1$/, '').replace(/\/v1$/, '') + fileUrl; if (type === 'character') { newCharacter.value.avatarUrl = fileUrl; characterFileList.value = [{ url: fileUrl, name: '角色图' }] } else { newScene.value.imageUrl = fileUrl; sceneFileList.value = [{ url: fileUrl, name: '场景图' }] } MessagePlugin.success('上传成功') } else { MessagePlugin.error(response?.msg || '上传失败') }
}


// ========== 8. 场景提取 & 分镜拆分 & 编辑 ==========
const handleExtractScenes = async () => {
    if (!currentScriptData.value?.id) return MessagePlugin.warning('剧本数据缺失')
    extractingScenes.value = true
    try {
        const res = await extractScenesTask({ scriptId: currentScriptData.value.id })
        const taskId = res.data?.data?.task_id || res.data?.taskId || res.data?.task_id
        if (taskId) {
            MessagePlugin.loading('AI 正在提取场景...')
            pollTask(taskId, () => { extractingScenes.value = false; MessagePlugin.success('场景提取完成'); loadScenes() }, () => extractingScenes.value = false)
        } else { MessagePlugin.error('提交失败'); extractingScenes.value = false }
    } catch { extractingScenes.value = false }
}

const generateShots = async () => {
    if (!currentScriptData.value?.id) return MessagePlugin.warning('请先保存剧本')

    generatingShots.value = true
    shotProcessPercent.value = 0 // 重置进度

    try {
        const res = await generateShotsTask({ scriptId: currentScriptData.value.id })
        const taskId = res.data?.data?.task_id || res.data?.taskId || res.data?.task_id

        if (taskId) {
            MessagePlugin.info('AI 正在拆分镜头，请稍候...')
            pollTask(
                taskId,
                () => {
                    generatingShots.value = false
                    shotProcessPercent.value = 100
                    MessagePlugin.success('分镜拆解完成')
                    loadShots(currentScriptData.value.id)
                },
                () => {
                    generatingShots.value = false
                    shotProcessPercent.value = 100 // 失败也重置，允许重新操作
                },
                (p) => {
                    shotProcessPercent.value = p // 🔴 更新实时进度
                }
            )
        } else {
            generatingShots.value = false
            shotProcessPercent.value = 100
        }
    } catch {
        generatingShots.value = false
        shotProcessPercent.value = 100
    }
}

// 隐藏弹窗避免卡顿
const regenerateShots = () => {
    const dialog = DialogPlugin.confirm({
        header: '重新拆分',
        body: '确定重新拆分吗？旧数据将被覆盖',
        onConfirm: () => {
            dialog.hide();
            generateShots()
        }
    })
}

const handleDeleteShotFromList = async (id: number) => {
    try {
        await deleteShots(id)
        MessagePlugin.success('删除成功')
        loadShots(currentScriptData.value.id)
    } catch { MessagePlugin.error('删除失败') }
}

// 分镜编辑
const openEditShotDialog = (row: any) => {
    // 深拷贝避免直接污染表格数据
    shotFormData.value = JSON.parse(JSON.stringify(row))

    // 将毫秒转换为秒提供给表单
    shotFormData.value.durationSec = (shotFormData.value.durationMs || 3000) / 1000

    shotDialog.visible = true
}

const handleShotSubmit = async () => {
    shotDialog.loading = true
    try {
        // 保存前，将秒转换为毫秒
        shotFormData.value.durationMs = shotFormData.value.durationSec * 1000

        // 构造提交负载，可过滤掉辅助字段
        const payload = { ...shotFormData.value }
        delete payload.durationSec

        const res = await updateShots(payload.id, payload)
        if (res.code === 0) {
            MessagePlugin.success('更新成功')
            shotDialog.visible = false
            loadShots(currentScriptData.value.id)
        } else {
            MessagePlugin.error('更新失败')
        }
    } catch {
        MessagePlugin.error('请求异常')
    } finally {
        shotDialog.loading = false
    }
}

// 🔴 表格列配置
const shotColumns = [
    { colKey: 'index', title: '#', width: 50, cell: (h: any, { row, rowIndex }: any) => row.sequenceNo || rowIndex + 1 },
    { colKey: 'sceneInfo', title: '场景/时间', width: 120, cell: 'sceneInfo' },
    { colKey: 'shotInfo', title: '镜头参数', width: 120, cell: 'shotInfo' },
    { colKey: 'visual', title: '画面与动作', minWidth: 220, cell: 'visual' },
    { colKey: 'dialogue', title: '台词', width: 130, ellipsis: true },
    { colKey: 'duration', title: '时长', width: 70, cell: 'duration' },
    { colKey: 'operation', title: '操作', width: 90, fixed: 'right', cell: 'operation' }
]

const nextStep = () => { if (currentStep.value < 2) currentStep.value++ }
const prevStep = () => { if (currentStep.value > 0) currentStep.value-- }
const goBack = () => router.back()

onMounted(() => initData())
</script>

<style scoped lang="less">
.workflow-container {
    min-height: 100vh;
    background: var(--td-bg-color-container-gray);
    display: flex;
    flex-direction: column;
}

.workflow-header {
    background: #fff;
    height: 64px;
    padding: 0 24px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
    flex-shrink: 0;
    z-index: 10;

    .header-left {
        display: flex;
        align-items: center;
        gap: 16px;
        width: 280px;

        .header-title {
            display: flex;
            align-items: center;
            gap: 8px;

            .title {
                font-weight: 700;
                font-size: 16px;
            }
        }
    }

    .header-center {
        flex: 1;
        display: flex;
        justify-content: center;
        align-items: center;
        height: 100%;
    }

    .header-right {
        width: 280px;
        display: flex;
        justify-content: flex-end;
    }
}

.full-height-card {
    height: 100%;
    display: flex;
    flex-direction: column;

    :deep(.t-card__body) {
        flex: 1;
        display: flex;
        flex-direction: column;
        overflow: hidden;
    }
}

.script-editor-container,
.script-preview-container {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 16px;
}

.script-textarea,
.readonly-textarea {
    flex: 1;

    :deep(textarea) {
        height: 100% !important;
        resize: none;
        font-family: monospace;
        line-height: 1.8;
    }
}

.readonly-textarea :deep(textarea) {
    background: var(--td-bg-color-secondarycontainer);
    border: none;
}

.preview-header,
.editor-toolbar {
    display: flex;
    justify-content: space-between;
    align-items: center;

    .ph-left {
        display: flex;
        gap: 12px;
        align-items: center;

        h3 {
            margin: 0;
        }

        .update-time {
            font-size: 12px;
            color: var(--td-text-color-secondary);
        }
    }
}

.editor-footer {
    display: flex;
    justify-content: flex-end;
}

.step-actions {
    display: flex;
    justify-content: center;
    gap: 16px;
    padding-top: 16px;
}

.empty-state-wrapper {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
}

.stage-area {
    flex: 1;
    padding: 24px;
    overflow-y: auto;
}

.stage-wrapper {
    height: 100%;
}

.card-header-flex {
    display: flex;
    justify-content: space-between;
    align-items: center;

    .header-info {
        display: flex;
        align-items: center;
        gap: 8px;

        .title {
            font-weight: 700;
            font-size: 16px;
        }

        .subtitle {
            color: var(--td-text-color-secondary);
            font-size: 12px;
        }
    }

    .header-actions {
        display: flex;
        gap: 12px;
    }
}

.toolbar-section {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px;
    background: var(--td-bg-color-secondarycontainer);
    border-radius: 6px;
    margin-bottom: 20px;

    .stat-text {
        margin-right: 16px;
        color: var(--td-text-color-secondary);
        font-size: 12px;
    }

    .toolbar-right {
        display: flex;
        align-items: center;
        gap: 12px;
    }
}

.character-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
    gap: 20px;
    overflow-y: auto;
    padding: 4px;
}

.char-card {
    background: #fff;
    border: 1px solid var(--td-border-level-1-color);
    border-radius: 8px;
    overflow: hidden;
    position: relative;
    transition: all 0.2s;
    display: flex;
    flex-direction: column;

    &:hover {
        transform: translateY(-2px);
        box-shadow: var(--td-shadow-2);

        .char-actions {
            opacity: 1;
        }
    }

    &.is-selected {
        border-color: var(--td-brand-color);
        background: var(--td-brand-color-light);
    }

    &.add-card {
        border: 2px dashed var(--td-component-stroke);
        display: flex;
        align-items: center;
        justify-content: center;
        cursor: pointer;
        min-height: 320px;

        &:hover {
            border-color: var(--td-brand-color);
            color: var(--td-brand-color);
        }

        .add-content {
            text-align: center;
            display: flex;
            flex-direction: column;
            align-items: center;
            gap: 8px;
        }
    }

    .card-select {
        position: absolute;
        top: 8px;
        right: 8px;
        z-index: 2;
    }

    .char-image {
        height: 200px;
        background: var(--td-bg-color-secondarycontainer);
        display: flex;
        align-items: center;
        justify-content: center;
        overflow: hidden;
        position: relative;

        .img-box {
            width: 100%;
            height: 100%;
        }

        .img-placeholder {
            color: var(--td-text-color-disabled);
        }

        .loading-mask {
            position: absolute;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background: rgba(255, 255, 255, 0.8);
            display: flex;
            align-items: center;
            justify-content: center;
            z-index: 5;
        }
    }

    .char-info {
        padding: 12px;
        flex: 1;

        .info-head {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 6px;

            .name {
                font-weight: 700;
                font-size: 14px;
            }
        }

        .desc {
            font-size: 12px;
            color: var(--td-text-color-secondary);
            height: 36px;
            margin-bottom: 8px;
        }
    }

    .char-actions {
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        padding: 8px;
        display: flex;
        gap: 8px;
        opacity: 0;
        transition: opacity 0.2s;
        background: linear-gradient(to bottom, rgba(0, 0, 0, 0.3), transparent);
    }
}

.image-upload-container {
    width: 100%;
}

.uploaded-images-container {
    display: flex;
    flex-direction: column;
    gap: 12px;
}

.upload-trigger {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 10px;
    height: 120px;
    width: 120px;
    border: 1px dashed var(--td-component-stroke);
    border-radius: 4px;
    cursor: pointer;
    background-color: var(--td-bg-color-container);
    transition: border-color 0.2s;

    &:hover {
        border-color: var(--td-brand-color);
    }

    .upload-text {
        margin-top: 8px;
        font-size: 12px;
        color: var(--td-text-color-secondary);
    }
}

.uploaded-item {
    margin-bottom: 8px;
}

.image-preview-wrapper {
    position: relative;
    width: 120px;
    height: 120px;
}

.image-preview {
    width: 100%;
    height: 100%;
    border-radius: 4px;
    border: 1px solid var(--td-component-stroke);
}

.image-overlay {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    justify-content: center;
    align-items: center;
    opacity: 0;
    transition: opacity 0.2s;
    border-radius: 4px;
}

.image-preview-wrapper:hover .image-overlay {
    opacity: 1;
}

.overlay-btn {
    color: #fff !important;
}

.reupload-component {
    margin-top: 4px;
}

.split-view {
    display: flex;
    height: 100%;
    gap: 24px;

    .scenes-panel {
        width: 340px;
        flex-shrink: 0;
        border-right: 1px solid var(--td-component-stroke);
        padding-right: 16px;
        overflow-y: auto;
        display: flex;
        flex-direction: column;
        gap: 12px;

        .panel-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            font-weight: 700;
            font-size: 14px;
            color: var(--td-text-color-primary);
        }

        .scene-batch-bar {
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 8px 12px;
            background: var(--td-bg-color-secondarycontainer);
            border-radius: 6px;
        }

        .scene-item {
            border: 1px solid var(--td-component-stroke);
            border-radius: 8px;
            background: #fff;
            overflow: hidden;
            transition: all 0.2s;
            display: flex;
            flex-direction: column;

            &:hover {
                border-color: var(--td-brand-color);
                box-shadow: var(--td-shadow-1);
            }

            &.is-selected {
                border-color: var(--td-brand-color);
                background: var(--td-brand-color-light);
            }

            .scene-image-area {
                height: 140px;
                position: relative;
                background: var(--td-bg-color-secondarycontainer);
                cursor: pointer;

                .s-img {
                    width: 100%;
                    height: 100%;
                }

                .img-placeholder {
                    height: 100%;
                    display: flex;
                    flex-direction: column;
                    gap: 4px;
                    align-items: center;
                    justify-content: center;
                    color: var(--td-text-color-disabled);

                    .placeholder-text {
                        font-size: 12px;
                    }

                    &:hover {
                        color: var(--td-brand-color);
                    }
                }

                .loading-mask {
                    position: absolute;
                    inset: 0;
                    background: rgba(255, 255, 255, 0.8);
                    display: flex;
                    align-items: center;
                    justify-content: center;
                    z-index: 2;
                }

                .scene-select-box {
                    position: absolute;
                    top: 6px;
                    left: 6px;
                    z-index: 3;
                }
            }

            .scene-content {
                padding: 10px 12px;
                flex: 1;

                .scene-header-row {
                    display: flex;
                    justify-content: space-between;
                    align-items: center;
                    margin-bottom: 6px;

                    .scene-name {
                        font-weight: 600;
                        font-size: 14px;
                    }
                }

                .scene-loc {
                    font-size: 12px;
                    color: var(--td-text-color-secondary);
                    margin-bottom: 4px;
                    display: flex;
                    align-items: center;
                    gap: 4px;
                }

                .scene-desc {
                    font-size: 12px;
                    color: var(--td-text-color-placeholder);
                    height: 36px;
                }
            }

            .scene-footer {
                border-top: 1px solid var(--td-component-stroke);
                background-color: var(--td-bg-color-container);

                :deep(.t-button) {
                    border-radius: 0;
                    height: 32px;

                    &:hover {
                        background-color: var(--td-bg-color-secondarycontainer);
                    }
                }

                :deep(.t-col:not(:last-child)) {
                    border-right: 1px solid var(--td-component-stroke);
                }
            }
        }
    }

    .shots-panel {
        flex: 1;
        overflow-y: auto;
        display: flex;
        flex-direction: column;

        &.full-width {
            width: 100%;
        }

        .panel-header {
            font-weight: 700;
            font-size: 14px;
            margin-bottom: 12px;
        }
    }
}

.mt-4 {
    margin-top: 16px;
}

.text-ellipsis-2 {
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
}

.text-ellipsis-2 {
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
}
</style>
