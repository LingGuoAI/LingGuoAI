<template>
    <div class="professional-editor">
        <div class="editor-header">
            <div class="header-left">
                <t-button variant="text" shape="circle" @click="goBack">
                    <template #icon><t-icon name="arrow-left" /></template>
                </t-button>
                <div class="header-title">
                    <span class="title">{{ project?.title || '加载中...' }}</span>
                    <t-tag theme="primary" variant="light" style="margin-left: 8px;">第 {{ episodeNumber }} 集</t-tag>
                </div>
            </div>
            <div class="header-right">
                <div class="status-text" v-if="saving"><t-loading size="small" /> 自动保存中...</div>
                <t-button theme="default" variant="outline" size="small" @click="loadData">
                    <template #icon><t-icon name="refresh" /></template> 刷新
                </t-button>
                <t-button theme="primary" size="small" @click="exportVideo" :loading="mergingVideo">
                    <template #icon><t-icon name="download" /></template> 导出视频
                </t-button>
            </div>
        </div>

        <div class="editor-main" v-loading="loading">
            <div class="left-sidebar">
                <div class="storyboard-panel">
                    <div class="panel-header">
                        <h3>分镜列表 ({{ storyboards.length }})</h3>
                        <t-button theme="primary" variant="text" size="small" @click="handleAddStoryboard">
                            <template #icon><t-icon name="add" /></template>新增
                        </t-button>
                    </div>
                    <div class="storyboard-list" v-if="storyboards.length > 0">
                        <div v-for="(shot, index) in storyboards" :key="shot.id" class="storyboard-item"
                            :class="{ active: String(currentStoryboardId) === String(shot.id) }"
                            @click="selectStoryboard(shot.id)" draggable="true"
                            @dragstart="handleDragStart($event, shot, 'storyboard')">
                            <div class="shot-thumb">
                                <t-image v-if="shot.image || shot.imageUrl"
                                    :src="getImageUrl(shot.image || shot.imageUrl)" fit="cover"
                                    style="width: 100%; height: 100%;" />
                                <div v-else class="drag-hint"><t-icon name="move" /></div>
                            </div>
                            <div class="shot-content">
                                <div class="shot-header">
                                    <div class="shot-title-row">
                                        <span class="shot-number">#{{ shot.sequenceNo || index + 1 }}</span>
                                        <span class="shot-title" :title="shot.title">{{ shot.title || '未命名镜头' }}</span>
                                    </div>
                                    <div class="shot-actions">
                                        <span class="shot-duration">{{ (shot.durationMs || 3000) / 1000 }}s</span>
                                        <t-popconfirm content="确认删除此镜头吗？" @confirm="handleDeleteStoryboard(shot)">
                                            <t-button shape="circle" size="small" theme="danger" variant="text"
                                                @click.stop>
                                                <template #icon><t-icon name="delete" /></template>
                                            </t-button>
                                        </t-popconfirm>
                                    </div>
                                </div>
                                <div class="shot-desc">{{ shot.visualDesc || shot.action || '暂无画面描述' }}</div>
                            </div>
                        </div>
                    </div>
                    <t-empty v-else description="暂无分镜，点击新增" style="padding: 20px 0" />
                </div>

                <div class="assets-panel">
                    <div class="panel-header assets-header">
                        <h3>素材库 ({{ videoAssets.length }})</h3>
                        <div class="header-actions">
                            <t-button theme="primary" variant="text" size="small" @click="addAllAssetsToTimeline">
                                <template #icon><t-icon name="add-rectangle" /></template>一键添加
                            </t-button>
                            <t-button theme="success" variant="outline" size="small" @click="exportVideo"
                                :disabled="timelineClips.length === 0" :loading="mergingVideo">
                                <template #icon><t-icon name="layers" /></template>合成当前剧集
                            </t-button>
                            <t-button theme="default" variant="text" size="small" @click="loadVideoAssets">
                                <template #icon><t-icon name="refresh" /></template>
                            </t-button>
                        </div>
                    </div>
                    <div class="assets-grid" v-if="videoAssets.length > 0">
                        <div v-for="asset in videoAssets" :key="asset.id" class="asset-item" draggable="true"
                            @dragstart="handleDragStart($event, asset, 'asset')">
                            <div class="asset-thumb">
                                <video :src="getImageUrl(asset.videoUrl || asset.video_url || asset.url)" muted
                                    @mouseenter="$event.target.play()" @mouseleave="$event.target.pause()"
                                    @loadedmetadata="$event.target.currentTime = 0"></video>
                                <span class="duration">{{ asset.duration || 5 }}s</span>

                                <div class="hover-overlay">
                                    <div class="icon-btn primary" @click.stop="addAssetToTimeline(asset)" title="添加到轨道">
                                        <t-icon name="add-circle" />
                                    </div>
                                    <div class="icon-btn danger" @click.stop="handleDeleteSource(asset)" title="删除素材">
                                        <t-icon name="delete" />
                                    </div>
                                </div>
                            </div>
                            <div class="asset-name" :title="asset.name">{{ asset.name || `分镜 ${asset.shotNumber ||
                                asset.shot_number || '-'} 素材` }}</div>
                        </div>
                    </div>
                    <t-empty v-else description="暂无素材，请在生成后添加" size="small" class="empty-assets" />
                </div>
            </div>

            <div class="center-workspace">
                <div class="preview-stage">
                    <div class="player-container">
                        <video v-if="currentPreviewUrl" ref="mainPlayerRef" :src="currentPreviewUrl" controls
                            class="main-player"></video>
                        <div v-else class="player-placeholder">
                            <t-icon name="film" size="48px" />
                            <p>请在时间线上选择片段或点击播放</p>
                        </div>
                    </div>
                </div>
                <div class="timeline-stage">
                    <VideoTimelineEditor ref="timelineEditorRef" :clips="timelineClips" :audio-clips="audioClips"
                        :current-time="currentTime" :total-duration="totalDuration" :current-id="currentStoryboardId"
                        @update:time="updateCurrentTime" @drop-clip="handleTimelineDrop"
                        @select-clip="handleTimelineSelect" @delete-clip="removeClipFromTimeline" />
                </div>
            </div>

            <div class="edit-panel">
                <t-tabs v-model="activeTab" theme="normal" class="edit-tabs">

                    <t-tab-panel value="clip" label="轨道片段">
                        <div class="tab-content scrollable-content" v-if="selectedTimelineClip">
                            <t-form label-align="top" class="compact-form">
                                <div class="section-group">
                                    <div class="section-header"><span>基本信息</span></div>
                                    <t-form-item label="起止时间 (秒)">
                                        <div style="display: flex; align-items: center; gap: 8px;">
                                            <t-input-number v-model="selectedTimelineClip.start" :min="0" :step="0.1"
                                                theme="normal" style="width: 100px" />
                                            <span>至</span>
                                            <t-input-number
                                                :value="Number((selectedTimelineClip.start + selectedTimelineClip.duration).toFixed(2))"
                                                disabled theme="normal" style="width: 100px" />
                                        </div>
                                    </t-form-item>
                                    <t-form-item label="片段时长 (秒)">
                                        <t-input-number v-model="selectedTimelineClip.duration" :min="0.1" :step="0.1"
                                            theme="normal" />
                                    </t-form-item>
                                    <t-form-item label="所在视频轨">
                                        <t-select v-model="selectedTimelineClip.track"
                                            :options="[{ label: '主视频轨', value: 0 }, { label: '画中画轨 (PIP)', value: 1 }]" />
                                    </t-form-item>
                                </div>
                                <t-divider />
                                <div class="section-group">
                                    <div class="section-header"><span>转场效果 (Transition)</span></div>
                                    <t-form-item label="进入转场">
                                        <t-select v-model="selectedTimelineClip.transition.type" :options="[
                                            { label: '无效果 (None)', value: 'none' },
                                            { label: '黑场过渡 (Fade)', value: 'fade' },
                                            { label: '交叉溶解 (Crossfade)', value: 'crossfade' },
                                            { label: '白场闪烁 (Flash)', value: 'flash' },
                                            { label: '拉近放大 (Zoom In)', value: 'zoom_in' }
                                        ]" />
                                    </t-form-item>
                                    <t-form-item label="转场时长 (秒)" v-if="selectedTimelineClip.transition.type !== 'none'"
                                        style="margin-top: 10px;">
                                        <t-slider v-model="selectedTimelineClip.transition.duration" :min="0.1" :max="3"
                                            :step="0.1" />
                                    </t-form-item>
                                </div>
                            </t-form>
                        </div>
                        <t-empty v-else description="请在底部时间线中选中一个片段" style="margin-top: 40px" />
                    </t-tab-panel>

                    <t-tab-panel value="shot" label="镜头属性">
                        <div class="tab-content scrollable-content" v-if="currentStoryboard">
                            <t-form label-align="top" class="compact-form">

                                <div class="section-group">
                                    <div class="section-header">
                                        <span>场景 (Scene)</span>
                                        <t-button theme="primary" variant="text" size="small"
                                            @click="showSceneSelector = true">更换场景</t-button>
                                    </div>
                                    <div class="scene-card" v-if="currentScene">
                                        <t-image-viewer v-if="currentScene.imageUrl" :close-on-overlay="true"
                                            :images="[getImageUrl(currentScene.imageUrl)]">
                                            <template #trigger="{ open }">
                                                <t-image :src="getImageUrl(currentScene.imageUrl)" fit="cover"
                                                    class="scene-cover" @click.stop="open" style="cursor: zoom-in;" lazy
                                                    error="图片加载失败" />
                                            </template>
                                        </t-image-viewer>

                                        <div class="scene-info">
                                            <div class="scene-loc">{{ currentScene.name }}</div>
                                            <div class="scene-meta">{{ currentScene.location }} · {{ currentScene.time
                                                }}</div>
                                        </div>
                                    </div>
                                    <div v-else class="empty-box" @click="showSceneSelector = true">
                                        <t-icon name="image" /> <span>点击关联场景</span>
                                    </div>
                                </div>

                                <div class="section-group">
                                    <div class="section-header">
                                        <span>登场角色 (Cast)</span>
                                        <t-button theme="primary" variant="text" size="small"
                                            @click="showCharacterSelector = true">
                                            <template #icon><t-icon name="add" /></template> 添加
                                        </t-button>
                                    </div>
                                    <div class="cast-list" v-if="selectedCharacters.length > 0">
                                        <div v-for="charId in selectedCharacters" :key="charId" class="cast-item">
                                            <t-image-viewer v-if="getCharacterById(charId)?.avatarUrl"
                                                :close-on-overlay="true"
                                                :images="[getImageUrl(getCharacterById(charId)?.avatarUrl)]">
                                                <template #trigger="{ open }">
                                                    <t-avatar :image="getImageUrl(getCharacterById(charId)?.avatarUrl)"
                                                        size="medium" shape="circle" @click.stop="open"
                                                        style="cursor: zoom-in;" />
                                                </template>
                                            </t-image-viewer>
                                            <t-avatar v-else size="medium" shape="circle">{{
                                                getCharacterById(charId)?.name?.[0] || '?' }}</t-avatar>

                                            <span class="cast-name" :title="getCharacterById(charId)?.name">{{
                                                getCharacterById(charId)?.name }}</span>
                                            <div class="remove-btn" @click="toggleCharacterInShot(charId)"><t-icon
                                                    name="close" /></div>
                                        </div>
                                    </div>
                                    <div v-else class="empty-text">暂无角色</div>
                                </div>

                                <div class="section-group">
                                    <div class="section-header">
                                        <span>相关道具 (Props)</span>
                                        <t-button theme="primary" variant="text" size="small"
                                            @click="showPropSelector = true">
                                            <template #icon><t-icon name="add" /></template> 添加
                                        </t-button>
                                    </div>
                                    <div class="cast-list" v-if="selectedProps.length > 0">
                                        <div v-for="propId in selectedProps" :key="propId" class="cast-item">
                                            <t-image-viewer v-if="getPropById(propId)?.imageUrl"
                                                :close-on-overlay="true"
                                                :images="[getImageUrl(getPropById(propId)?.imageUrl)]">
                                                <template #trigger="{ open }">
                                                    <t-image :src="getImageUrl(getPropById(propId)?.imageUrl)"
                                                        fit="contain"
                                                        style="width: 40px; height: 40px; border-radius: 4px; background: #eee; cursor: zoom-in;"
                                                        @click.stop="open" lazy error="加载失败" />
                                                </template>
                                            </t-image-viewer>
                                            <t-icon v-else name="image" size="24px" style="color: #ccc" />

                                            <span class="cast-name" :title="getPropById(propId)?.name">{{
                                                getPropById(propId)?.name }}</span>
                                            <div class="remove-btn" @click="togglePropInShot(propId)"><t-icon
                                                    name="close" />
                                            </div>
                                        </div>
                                    </div>
                                    <div v-else class="empty-text">暂无道具</div>
                                </div>

                                <t-divider />

                                <div class="section-group">
                                    <div class="section-header"><span>视效设置</span></div>
                                    <t-row :gutter="10">
                                        <t-col :span="6">
                                            <t-form-item label="景别">
                                                <t-select v-model="currentStoryboard.shotType" size="small"
                                                    :options="['大远景', '远景', '全景', '中景', '近景', '特写', '大特写'].map(v => ({ label: v, value: v }))"
                                                    @change="saveStoryboardField" />
                                            </t-form-item>
                                        </t-col>
                                        <t-col :span="6">
                                            <t-form-item label="视角">
                                                <t-select v-model="currentStoryboard.angle" size="small"
                                                    :options="['平视', '俯视', '仰视', '侧视', '航拍'].map(v => ({ label: v, value: v }))"
                                                    @change="saveStoryboardField" />
                                            </t-form-item>
                                        </t-col>
                                    </t-row>
                                    <t-form-item label="运镜" style="margin-top: 10px;">
                                        <t-select v-model="currentStoryboard.cameraMovement" size="small"
                                            :options="['固定镜头', '推镜', '拉镜', '摇镜', '移镜', '跟镜', '环绕'].map(v => ({ label: v, value: v }))"
                                            @change="saveStoryboardField" />
                                    </t-form-item>
                                    <t-form-item label="时长 (秒)" style="margin-top: 10px;">
                                        <t-slider :value="(currentStoryboard.durationMs || 3000) / 1000" :min="1"
                                            :max="60" @change="updateShotDurationMs" />
                                    </t-form-item>
                                </div>

                                <div class="section-group">
                                    <div class="section-header"><span>叙事内容</span></div>
                                    <t-form-item label="动作 (Action)">
                                        <t-textarea v-model="currentStoryboard.action" :autosize="{ minRows: 2 }"
                                            placeholder="角色做的动作..." @blur="saveStoryboardField" />
                                    </t-form-item>
                                    <t-form-item label="结果 (Result)">
                                        <t-textarea v-model="currentStoryboard.result" :autosize="{ minRows: 2 }"
                                            placeholder="动作导致的结果..." @blur="saveStoryboardField" />
                                    </t-form-item>
                                    <t-form-item label="对白 (Dialogue)">
                                        <t-textarea v-model="currentStoryboard.dialogue" :autosize="{ minRows: 2 }"
                                            placeholder="角色台词..." @blur="saveStoryboardField" />
                                    </t-form-item>
                                    <t-form-item label="画面描述 (Visual)">
                                        <t-textarea v-model="currentStoryboard.visualDesc" :autosize="{ minRows: 3 }"
                                            placeholder="详细的画面描述..." @blur="saveStoryboardField" />
                                    </t-form-item>
                                    <t-form-item label="氛围 (Atmosphere)">
                                        <t-textarea v-model="currentStoryboard.atmosphere" :autosize="{ minRows: 2 }"
                                            placeholder="光影、色调、气氛..." @blur="saveStoryboardField" />
                                    </t-form-item>
                                </div>

                                <t-divider />

                                <div class="section-group">
                                    <div class="section-header"><span>音频设置</span></div>
                                    <t-form-item label="音效与配乐 (Audio Prompt)">
                                        <t-textarea v-model="currentStoryboard.audioPrompt" :autosize="{ minRows: 2 }"
                                            placeholder="例如：开门声、脚步声、悲伤的钢琴曲..." @blur="saveStoryboardField" />
                                    </t-form-item>
                                </div>

                            </t-form>
                        </div>
                        <t-empty v-else description="请在左侧选择一个镜头" style="margin-top: 40px" />
                    </t-tab-panel>

                    <t-tab-panel value="image" label="镜头图片">
                        <div class="tab-content scrollable-content" v-if="currentStoryboard">
                            <div class="section-group">
                                <div class="section-header"><span>帧类型选择</span></div>
                                <t-radio-group variant="default-filled" v-model="selectedFrameType"
                                    style="width: 100%;">
                                    <t-radio-button value="first">首帧</t-radio-button>
                                    <t-radio-button value="last">尾帧</t-radio-button>
                                    <t-radio-button value="action">动作序列</t-radio-button>
                                    <t-radio-button value="key">关键帧</t-radio-button>
                                </t-radio-group>
                            </div>

                            <div class="section-group">
                                <div class="section-header">
                                    <span>AI 绘画提示词</span>
                                    <t-button theme="primary" variant="text" size="small" :loading="extractingPrompt"
                                        @click="extractFramePrompt">
                                        提取提示词
                                    </t-button>
                                </div>
                                <t-textarea v-model="currentFramePromptText" :rows="4" placeholder="输入英文提示词..."
                                    @blur="saveStoryboardField" />
                            </div>

                            <div class="action-bar">
                                <t-button theme="primary" :loading="generatingImage" @click="generateFrameImage">
                                    <template #icon><t-icon name="magic" /></template> 生成画面
                                </t-button>
                                <t-upload theme="custom" :action="uploadConfig.action" :headers="uploadConfig.headers"
                                    :show-file-list="false" accept="image/*" :before-upload="beforeUpload"
                                    @success="handleUploadImageSuccess" @fail="handleUploadFail">
                                    <t-button variant="outline" :loading="uploadingImage">
                                        <template #icon><t-icon name="upload" /></template>上传
                                    </t-button>
                                </t-upload>
                            </div>

                            <div class="section-group" style="margin-top: 20px;">
                                <div class="section-header"><span>生成结果 ({{ currentFrameImages.length }})</span></div>

                                <div v-if="selectedFrameType === 'action'" class="grid-entry-card"
                                    @click="showGridEditor = true">
                                    <t-icon name="add" size="24px" />
                                    <span>创建动作序列 (宫格图)</span>
                                </div>

                                <div class="image-grid-list" v-if="currentFrameImages.length > 0">
                                    <div v-for="img in currentFrameImages" :key="img.id" class="image-grid-item">
                                        <t-image :src="getImageUrl(img.url || img.imageUrl)" fit="cover" class="img" />

                                        <div class="img-overlay">
                                            <div class="actions-wrapper">
                                                <t-image-viewer :close-on-overlay="true"
                                                    :images="[getImageUrl(img.url || img.imageUrl)]">
                                                    <template #trigger="{ open }">
                                                        <div class="icon-btn" @click.stop="open" title="预览大图">
                                                            <t-icon name="zoom-in" size="18px" style="color: #fff;" />
                                                        </div>
                                                    </template>
                                                </t-image-viewer>
                                                <div class="icon-btn danger" @click.stop="deleteImage(img)"
                                                    title="删除图片">
                                                    <t-icon name="delete" size="18px" style="color: #fff;" />
                                                </div>
                                            </div>
                                            <div class="crop-btn" v-if="selectedFrameType === 'action'"
                                                @click.stop="openCropDialog(img)" title="前往裁剪九宫格">
                                                <t-icon name="cut" size="14px" />
                                            </div>
                                        </div>
                                    </div>
                                </div>
                                <div v-else-if="selectedFrameType !== 'action'" class="empty-text">暂无图片</div>
                            </div>
                        </div>
                        <t-empty v-else description="请选择一个镜头" style="margin-top: 40px" />
                    </t-tab-panel>

                    <t-tab-panel value="video" label="视频生成">
                        <div class="tab-content scrollable-content" v-if="currentStoryboard">

                            <div class="video-prompt-box">
                                {{ currentStoryboard.videoPrompt || currentStoryboard.imagePrompt ||
                                    currentStoryboard.visualDesc || '暂无提示词' }}
                            </div>

                            <div class="video-settings section-group">
                                <t-form-item label="视频模型">
                                    <t-select v-model="selectedVideoModel" placeholder="请选择视频模型">
                                        <t-option v-for="model in videoModelCapabilities" :key="model.id"
                                            :value="model.id" :label="model.name">
                                            <div
                                                style="display: flex; justify-content: space-between; align-items: center;">
                                                <span>{{ model.name }}</span>
                                                <div class="model-tags">
                                                    <t-tag v-if="model.supportMultipleImages" size="small"
                                                        theme="success" variant="light"
                                                        style="margin-left: 4px">多图</t-tag>
                                                    <t-tag v-if="model.supportFirstLastFrame" size="small"
                                                        theme="primary" variant="light"
                                                        style="margin-left: 4px">首尾帧</t-tag>
                                                    <t-tag size="small" theme="default" variant="light"
                                                        style="margin-left: 4px">最多{{ model.maxImages }}张</t-tag>
                                                </div>
                                            </div>
                                        </t-option>
                                    </t-select>
                                </t-form-item>

                                <t-alert theme="info"
                                    :message="`视频时长使用镜头属性：${currentShotDurationSeconds} 秒。如需修改，请到“镜头属性”调整。`" />

                                <t-form-item label="参考图模式"
                                    v-if="selectedVideoModel && availableReferenceModes.length > 0">
                                    <t-select v-model="referenceMode" placeholder="请选择参考图模式">
                                        <t-option v-for="mode in availableReferenceModes" :key="mode.value"
                                            :value="mode.value" :label="mode.label">
                                            <div
                                                style="display: flex; justify-content: space-between; align-items: center;">
                                                <span>{{ mode.label }}</span>
                                                <span v-if="mode.description"
                                                    style="color: var(--td-text-color-placeholder); font-size: 12px;">{{
                                                        mode.description }}</span>
                                            </div>
                                        </t-option>
                                    </t-select>
                                </t-form-item>

                                <div class="reference-config-section" v-if="referenceMode && referenceMode !== 'none'">

                                    <div class="image-slots-container">
                                        <div v-if="referenceMode === 'single'" style="text-align: center">
                                            <div class="reference-mode-title">单图参考</div>
                                            <div style="display: inline-block">
                                                <t-upload theme="custom" :action="uploadConfig.action"
                                                    :headers="uploadConfig.headers" :show-file-list="false"
                                                    accept="image/*" :before-upload="beforeUpload"
                                                    @success="(ctx) => handleUploadRefSuccess(ctx, 'single')">
                                                    <div class="image-slot" :class="{ selected: !!singleRefImage }">
                                                        <t-image v-if="singleRefImage"
                                                            :src="getImageUrl(singleRefImage.url || singleRefImage.imageUrl)"
                                                            fit="cover" class="img" />
                                                        <div v-else class="image-slot-placeholder">
                                                            <t-icon name="add" size="24px" />
                                                            <div class="slot-hint">点击上传图片</div>
                                                        </div>
                                                    </div>
                                                </t-upload>
                                                <div class="image-slot-remove" v-if="singleRefImage"
                                                    @click.stop="removeSelectedImage(singleRefImage.id)">
                                                    <t-icon name="close" size="14px" />
                                                </div>
                                            </div>
                                        </div>

                                        <div v-else-if="referenceMode === 'first_last'" class="first-last-slots">
                                            <div class="slot-wrapper">
                                                <div class="ref-label">首帧 (起点)</div>
                                                <div class="ref-image-wrapper">
                                                    <t-upload theme="custom" :action="uploadConfig.action"
                                                        :headers="uploadConfig.headers" :show-file-list="false"
                                                        accept="image/*" :before-upload="beforeUpload"
                                                        @success="(ctx) => handleUploadRefSuccess(ctx, 'first')">
                                                        <div class="ref-image-slot"
                                                            :class="{ selected: !!firstRefImage }">
                                                            <t-image v-if="firstRefImage"
                                                                :src="getImageUrl(firstRefImage.url || firstRefImage.imageUrl)"
                                                                fit="cover" class="img" />
                                                            <div v-else class="placeholder">
                                                                <t-icon name="add" size="24px" />
                                                                <div class="slot-hint">点击上传首帧</div>
                                                            </div>
                                                        </div>
                                                    </t-upload>
                                                    <div class="ref-delete-btn" v-if="firstRefImage"
                                                        @click.stop="removeSelectedImage(firstRefImage.id)">
                                                        <t-icon name="close-circle-filled" size="18px" />
                                                    </div>
                                                </div>
                                            </div>

                                            <div class="slot-divider">
                                                <t-icon name="arrow-right" size="24px" />
                                            </div>

                                            <div class="slot-wrapper">
                                                <div class="ref-label">尾帧 (终点)</div>
                                                <div class="ref-image-wrapper">
                                                    <t-upload theme="custom" :action="uploadConfig.action"
                                                        :headers="uploadConfig.headers" :show-file-list="false"
                                                        accept="image/*" :before-upload="beforeUpload"
                                                        @success="(ctx) => handleUploadRefSuccess(ctx, 'last')">
                                                        <div class="ref-image-slot"
                                                            :class="{ selected: !!lastRefImage }">
                                                            <t-image v-if="lastRefImage"
                                                                :src="getImageUrl(lastRefImage.url || lastRefImage.imageUrl)"
                                                                fit="cover" class="img" />
                                                            <div v-else class="placeholder">
                                                                <t-icon name="add" size="24px" />
                                                                <div class="slot-hint">点击上传尾帧</div>
                                                            </div>
                                                        </div>
                                                    </t-upload>
                                                    <div class="ref-delete-btn" v-if="lastRefImage"
                                                        @click.stop="removeSelectedImage(lastRefImage.id)">
                                                        <t-icon name="close-circle-filled" size="18px" />
                                                    </div>
                                                </div>
                                            </div>
                                        </div>

                                        <div v-else-if="referenceMode === 'multiple'" class="ref-container-column">
                                            <div class="ref-label" style="text-align: center; margin-bottom: 12px;">已选多图
                                                (最大 {{
                                                    currentModelCapability?.maxImages || 6 }} 张)</div>
                                            <div
                                                style="display: flex; gap: 12px; justify-content: center; flex-wrap: wrap;">
                                                <div v-for="index in (currentModelCapability?.maxImages || 6)"
                                                    :key="index" style="position: relative;">
                                                    <t-upload theme="custom" :action="uploadConfig.action"
                                                        :headers="uploadConfig.headers" :show-file-list="false"
                                                        accept="image/*" :before-upload="beforeUpload"
                                                        @success="(ctx) => handleUploadRefSuccess(ctx, `multi_${index}`)">
                                                        <div class="image-slot image-slot-small"
                                                            :class="{ selected: !!getMultiRefImage(index) }">
                                                            <t-image v-if="getMultiRefImage(index)"
                                                                :src="getImageUrl(getMultiRefImage(index).url || getMultiRefImage(index).imageUrl)"
                                                                fit="cover" class="img" />
                                                            <div v-else class="image-slot-placeholder">
                                                                <t-icon name="add" size="16px" />
                                                                <div style="margin-top: 4px; font-size: 10px;">图 {{
                                                                    index }}
                                                                </div>
                                                            </div>
                                                        </div>
                                                    </t-upload>
                                                    <div class="image-slot-remove" v-if="getMultiRefImage(index)"
                                                        @click.stop="removeSelectedImage(getMultiRefImage(index).id)">
                                                        <t-icon name="close" size="12px" />
                                                    </div>
                                                </div>
                                            </div>
                                        </div>
                                    </div>

                                    <div class="reference-images-section">
                                        <div class="frame-type-buttons" v-if="referenceMode === 'first_last'"
                                            style="text-align: center; margin-bottom: 12px;">
                                            <t-radio-group v-model="selectedVideoFrameType" variant="default-filled">
                                                <t-radio-button value="first">首帧</t-radio-button>
                                                <t-radio-button value="last">尾帧</t-radio-button>
                                                <t-radio-button value="action">动作序列</t-radio-button>
                                                <t-radio-button value="key">关键帧</t-radio-button>
                                            </t-radio-group>
                                        </div>

                                        <div class="frame-type-content">
                                            <div v-if="referenceMode === 'first_last' && selectedVideoFrameType === 'first' && previousStoryboardLastFrames.length > 0"
                                                class="previous-frame-section">
                                                <div
                                                    style="display: flex; align-items: center; gap: 6px; margin-bottom: 6px;">
                                                    <t-tag size="small" theme="primary" variant="light">上一镜头 #{{
                                                        previousStoryboard?.sequenceNo }} 尾帧</t-tag>
                                                    <span class="hint-text">点击添加为首帧参考</span>
                                                </div>
                                                <div class="reference-grid">
                                                    <div v-for="img in previousStoryboardLastFrames"
                                                        :key="'prev-' + img.id" class="reference-item-mini"
                                                        :class="{ selected: isPreviousFrameSelected(img) }"
                                                        @click="handlePreviousImageSelect(img)">
                                                        <t-image :src="getImageUrl(img.url || img.imageUrl)" fit="cover"
                                                            class="img" />
                                                        <div v-if="isPreviousFrameSelected(img)" class="check-mark">✓
                                                        </div>
                                                    </div>
                                                </div>
                                            </div>

                                            <div class="image-scroll-container">
                                                <div class="reference-grid"
                                                    v-if="currentFrameImagesFiltered(selectedVideoFrameType).length > 0">
                                                    <div v-for="img in currentFrameImagesFiltered(selectedVideoFrameType)"
                                                        :key="img.id" class="reference-item-mini"
                                                        :class="{ selected: isImageSelected(img) }"
                                                        @click="handleImageSelect(img)">
                                                        <t-image :src="getImageUrl(img.url || img.imageUrl)" fit="cover"
                                                            class="img" />
                                                        <div v-if="isImageSelected(img)" class="check-mark">✓</div>
                                                        <t-image-viewer :close-on-overlay="true"
                                                            :images="[getImageUrl(img.url || img.imageUrl)]">
                                                            <template #trigger="{ open }">
                                                                <div class="preview-icon" @click.stop="open">
                                                                    <t-icon name="zoom-in" size="14px" />
                                                                </div>
                                                            </template>
                                                        </t-image-viewer>
                                                    </div>
                                                </div>
                                                <div v-else class="empty-text" style="padding: 20px 0;">
                                                    <t-empty
                                                        :description="referenceMode === 'first_last' ? `暂无${getFrameTypeName(selectedVideoFrameType)}图片，请在上方上传或在[镜头图片]面板生成` : '当前镜头暂无可用图片'" />
                                                </div>
                                            </div>
                                        </div>
                                    </div>
                                </div>

                                <t-button theme="primary" block size="large" :loading="generatingVideo"
                                    @click="generateVideo" style="margin-top: 24px;">
                                    <template #icon><t-icon name="video" /></template> 生成视频
                                </t-button>
                            </div>

                            <div class="video-list-area section-group" v-if="generatedVideos.length > 0">
                                <div class="section-header">
                                    <span>生成结果 ({{ generatedVideos.length }})</span>
                                </div>
                                <div class="video-card-list">
                                    <div v-for="video in generatedVideos" :key="video.id" class="video-card">

                                        <template
                                            v-if="video.status === 'completed' || video.status === 'succeeded' || video.status === 2">
                                            <div class="video-thumbnail" @click="playVideo(video)">
                                                <video
                                                    :src="getImageUrl(video.url || video.videoUrl || video.video_url)"
                                                    preload="metadata"></video>
                                                <div class="play-overlay">
                                                    <t-icon name="play-circle" size="40px"
                                                        style="color: white; opacity: 0.8;" />
                                                </div>
                                            </div>
                                        </template>
                                        <template v-else>
                                            <div class="video-placeholder">
                                                <t-loading v-if="video.status === 'processing' || video.status === 1"
                                                    size="small" />
                                                <t-icon v-else name="time" size="24px" style="color: #999;" />
                                                <p style="margin-top: 8px; font-size: 12px; color: #666;">
                                                    {{ getStatusText(video.status) }}
                                                </p>
                                            </div>
                                        </template>

                                        <div class="video-actions">
                                            <t-tag
                                                :theme="video.status === 'completed' || video.status === 'succeeded' || video.status === 2 ? 'success' : (video.status === 'failed' || video.status === 3 ? 'danger' : 'warning')"
                                                variant="light" size="small">
                                                {{ getStatusText(video.status) }}
                                            </t-tag>
                                            <div class="action-btns">
                                                <t-tooltip content="添加到素材库"
                                                    v-if="video.status === 'completed' || video.status === 'succeeded' || video.status === 2">
                                                    <t-button size="small" variant="text"
                                                        @click="addVideoToAssets(video)">
                                                        <t-icon name="layers" />
                                                    </t-button>
                                                </t-tooltip>
                                                <t-button size="small" variant="text" theme="danger"
                                                    @click="deleteVideo(video)"><t-icon name="delete" /></t-button>
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </div>
                        <t-empty v-else description="请选择一个镜头" style="margin-top: 40px" />
                    </t-tab-panel>

                    <t-tab-panel value="audio" label="音效配乐">
                        <div class="tab-content">
                            <t-empty description="音效与配乐生成功能开发中... 请在'镜头属性'中配置描述" />
                        </div>
                    </t-tab-panel>

                    <t-tab-panel value="merge" label="视频合成">
                        <div class="tab-content scrollable-content">
                            <div class="section-group">
                                <div class="section-header"><span>合成记录</span></div>
                                <div class="merge-list" v-if="videoMerges.length > 0">
                                    <div v-for="merge in videoMerges" :key="merge.id" class="merge-item">
                                        <div class="merge-info">
                                            <div class="title">{{ merge.title || '合成视频' }}</div>
                                            <div class="time">{{ merge.createTime || merge.created_at || '刚刚' }}</div>
                                        </div>
                                        <t-tag
                                            :theme="merge.status === 'completed' || merge.status === 2 ? 'success' : (merge.status === 'failed' || merge.status === 3 ? 'danger' : 'warning')">{{
                                                merge.status === 'completed' || merge.status === 2 ? '已完成' : (merge.status
                                                    ===
                                                    'failed' || merge.status === 3 ? '失败' : '处理中') }}</t-tag>
                                        <div style="display: flex; gap: 4px;">
                                            <t-button v-if="merge.url || merge.mergedUrl || merge.merged_url"
                                                size="small" variant="text"
                                                @click="previewImage(merge.url || merge.mergedUrl || merge.merged_url)">预览</t-button>
                                            <t-button size="small" variant="text" theme="danger"
                                                @click="deleteMergeItem(merge.id)"><t-icon name="delete" /></t-button>
                                        </div>
                                    </div>
                                </div>
                                <t-empty v-else description="暂无合成记录" size="small" />
                            </div>

                            <t-button theme="primary" block @click="exportVideo" size="large" :loading="mergingVideo">
                                <template #icon><t-icon name="layers" /></template> 开始合成当前时间线
                            </t-button>
                        </div>
                    </t-tab-panel>
                </t-tabs>
            </div>
        </div>

        <t-dialog v-model:visible="showSceneSelector" header="关联场景" width="500px">
            <t-list :split="true" style="max-height: 400px; overflow-y: auto">
                <t-list-item v-for="scene in sceneList" :key="scene.id" @click="linkSceneToShot(scene)"
                    style="cursor: pointer">
                    <t-list-item-meta :title="scene.name" :description="`${scene.location} · ${scene.time}`">
                        <template #image>
                            <t-image-viewer v-if="scene.imageUrl" :close-on-overlay="true"
                                :images="[getImageUrl(scene.imageUrl)]">
                                <template #trigger="{ open }">
                                    <t-image :src="getImageUrl(scene.imageUrl)" fit="cover"
                                        style="width: 50px; height: 50px; border-radius: 4px; cursor: zoom-in;"
                                        @click.stop="open" lazy error="加载失败" />
                                </template>
                            </t-image-viewer>
                            <t-icon v-else name="image" size="24px" style="color: #ccc" />
                        </template>
                    </t-list-item-meta>
                    <template #action>
                        <t-icon v-if="currentStoryboard?.sceneId === scene.id" name="check"
                            style="color: var(--td-brand-color)" />
                    </template>
                </t-list-item>
                <t-empty v-if="sceneList.length === 0" description="暂无场景数据" />
            </t-list>
        </t-dialog>

        <t-dialog v-model:visible="showCharacterSelector" header="选择角色" width="500px"
            @confirm="showCharacterSelector = false">
            <div class="char-selector-grid">
                <div v-for="char in availableCharacters" :key="char.id" class="char-item"
                    :class="{ selected: selectedCharacters.includes(char.id) }" @click="toggleCharacterInShot(char.id)">
                    <t-image-viewer v-if="char.avatarUrl" :close-on-overlay="true"
                        :images="[getImageUrl(char.avatarUrl)]">
                        <template #trigger="{ open }">
                            <t-avatar :image="getImageUrl(char.avatarUrl)" size="large" @click.stop="open"
                                style="cursor: zoom-in;" />
                        </template>
                    </t-image-viewer>
                    <t-avatar v-else size="large">{{ char.name ? char.name[0] : '?' }}</t-avatar>
                    <span>{{ char.name }}</span>
                    <div class="check" v-if="selectedCharacters.includes(char.id)"><t-icon name="check" /></div>
                </div>
            </div>
            <t-empty v-if="availableCharacters.length === 0" description="暂无角色" />
        </t-dialog>

        <t-dialog v-model:visible="showPropSelector" header="选择道具" width="500px" @confirm="showPropSelector = false">
            <div class="char-selector-grid">
                <div v-for="prop in availableProps" :key="prop.id" class="char-item"
                    :class="{ selected: selectedProps.includes(prop.id) }" @click="togglePropInShot(prop.id)">
                    <t-image-viewer v-if="prop.imageUrl" :close-on-overlay="true"
                        :images="[getImageUrl(prop.imageUrl)]">
                        <template #trigger="{ open }">
                            <t-image :src="getImageUrl(prop.imageUrl)" fit="contain"
                                style="width: 50px; height: 50px; border-radius: 4px; background: #f9f9f9; cursor: zoom-in;"
                                @click.stop="open" lazy error="加载失败" />
                        </template>
                    </t-image-viewer>
                    <t-icon v-else name="image" size="24px" style="color: #ccc" />
                    <span>{{ prop.name }}</span>
                    <div class="check" v-if="selectedProps.includes(prop.id)"><t-icon name="check" /></div>
                </div>
            </div>
            <t-empty v-if="availableProps.length === 0" description="暂无道具" />
        </t-dialog>

        <GridImageEditor v-model="showGridEditor" :storyboard-id="currentStoryboardId" :drama-id="dramaId"
            :all-images="currentFrameImages" @success="handleGridSuccess" />

        <ImageCropDialog v-model="showCropDialog" :image-url="cropImageUrl" @save="handleCropSave" />
    </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive, onMounted, watch, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { MessagePlugin, DialogPlugin } from 'tdesign-vue-next'

// API
import { findProjects } from '@/api/projects'
import { getAiConfigList } from '@/api/ai_config'
import { getScriptsList } from '@/api/scripts'
import { getScenesList } from '@/api/scenes'
import { getCharactersList } from '@/api/characters'
import { getPropsList } from '@/api/props'
import { getShotsList, createShots, updateShots, deleteShots } from '@/api/shots'
import { createSource, deleteSource } from '@/api/source'
import { extractFramePromptTask, findTasks, generateImageByPromptTask, generateVideoTask, mergeVideoTask } from '@/api/tasks'
import { createShotFrameImages, deleteShotFrameImages } from '@/api/shot_frame_image'
import { getImageUrl } from '@/utils/format'
import { request } from '@/utils/request'

// 组件
import VideoTimelineEditor from '@/components/editor/VideoTimelineEditor.vue'
import GridImageEditor from '@/components/editor/GridImageEditor.vue'
import ImageCropDialog from '@/components/editor/ImageCropDialog.vue'

const route = useRoute()
const router = useRouter()

// === 数据状态 ===
const loading = ref(false)
const saving = ref(false)
const dramaId = route.params.dramaId as string
const episodeNumber = Number(route.params.episodeNumber)

const project = ref<any>({})
const currentScriptId = ref<number | null>(null)
const storyboards = ref<any[]>([])
const currentStoryboardId = ref<string | number | null>(null)
const sceneList = ref<any[]>([])
const availableCharacters = ref<any[]>([])
const availableProps = ref<any[]>([])
const videoAssets = ref<any[]>([])
const timelineClips = ref<any[]>([])
const audioClips = ref<any[]>([])
const currentTime = ref(0)
const totalDuration = ref(60)

// 记录当前在时间线上选中的片段
const selectedTimelineClip = ref<any>(null)

// === 右侧面板状态 ===
const activeTab = ref('shot')
const showSceneSelector = ref(false)
const showCharacterSelector = ref(false)
const showPropSelector = ref(false)

// 图片生成状态
const selectedFrameType = ref('first')
const generatingImage = ref(false)
const extractingPrompt = ref(false)
const showGridEditor = ref(false)
const showCropDialog = ref(false)
const cropImageUrl = ref('')
const uploadingImage = ref(false)

// 合成状态
const videoMerges = ref<any[]>([])
const mergingVideo = ref(false)

const currentPreviewUrl = ref('')
const timelineEditorRef = ref<any>(null)
const mainPlayerRef = ref<HTMLVideoElement | null>(null)

// ================= 视频生成相关逻辑 =================
const generatingVideo = ref(false)
const selectedVideoModel = ref('doubao-seedance-1-5-pro-251215')
const currentShotDurationSeconds = computed(() => Math.max(1, Math.round(Number(currentStoryboard.value?.durationMs || 3000) / 1000)))
const referenceMode = ref('single')
const selectedVideoFrameType = ref('first')
const generatedVideos = ref<any[]>([])

const selectedImagesForVideo = ref<number[]>([])
const selectedLastImageForVideo = ref<number | null>(null)

interface VideoModelCapability {
    id: string;
    name: string;
    supportMultipleImages: boolean;
    supportFirstLastFrame: boolean;
    supportSingleImage: boolean;
    supportTextOnly: boolean;
    maxImages: number;
}

// 定义模型能力映射表，用于匹配数据库中导出的模型
const MODEL_CAPABILITY_MAP: Record<string, any> = {
    'sora-2': { name: 'OpenAI Sora', supportSingleImage: true, supportMultipleImages: false, supportFirstLastFrame: false, supportTextOnly: true, maxImages: 1 },
    'doubao-seedance-1-5-pro-251215': { name: '豆包 (Seedance)', supportSingleImage: true, supportMultipleImages: true, supportFirstLastFrame: true, supportTextOnly: true, maxImages: 6 },
    'MiniMax-Hailuo-02': { name: '海螺 (MiniMax)', supportSingleImage: true, supportMultipleImages: false, supportFirstLastFrame: true, supportTextOnly: true, maxImages: 2 },
    'kling': { name: '可灵 (Kling)', supportSingleImage: true, supportMultipleImages: false, supportFirstLastFrame: true, supportTextOnly: true, maxImages: 2 },
    'runway': { name: 'Runway Gen-3', supportSingleImage: true, supportMultipleImages: false, supportFirstLastFrame: true, supportTextOnly: true, maxImages: 2 },
    'pika': { name: 'Pika 1.0', supportSingleImage: true, supportMultipleImages: true, supportFirstLastFrame: false, supportTextOnly: true, maxImages: 4 },
    'veo-3.1-fast-generate-001': { name: 'Google Veo', supportSingleImage: true, supportMultipleImages: false, supportFirstLastFrame: false, supportTextOnly: true, maxImages: 1 }
};

// 默认兜底模型列表
const defaultVideoModels: VideoModelCapability[] = [
    { id: 'sora-2', name: 'OpenAI Sora', supportSingleImage: true, supportMultipleImages: false, supportFirstLastFrame: false, supportTextOnly: true, maxImages: 1 },
    { id: 'doubao-seedance-1-5-pro-251215', name: '豆包 (Seedance)', supportSingleImage: true, supportMultipleImages: true, supportFirstLastFrame: true, supportTextOnly: true, maxImages: 6 },
    { id: 'MiniMax-Hailuo-02', name: '海螺 (MiniMax)', supportSingleImage: true, supportMultipleImages: false, supportFirstLastFrame: true, supportTextOnly: true, maxImages: 2 },
    { id: 'kling', name: '可灵 (Kling)', supportSingleImage: true, supportMultipleImages: false, supportFirstLastFrame: true, supportTextOnly: true, maxImages: 2 },
    { id: 'runway', name: 'Runway Gen-3', supportSingleImage: true, supportMultipleImages: false, supportFirstLastFrame: true, supportTextOnly: true, maxImages: 2 },
    { id: 'pika', name: 'Pika 1.0', supportSingleImage: true, supportMultipleImages: true, supportFirstLastFrame: false, supportTextOnly: true, maxImages: 4 },
    {
        id: 'veo-3.1-fast-generate-001',
        name: 'Google Veo (Vertex AI)',
        supportSingleImage: true,       // 支持首帧图生视频
        supportMultipleImages: false,   // 暂不支持多图
        supportFirstLastFrame: false,   // 暂不支持首尾帧
        supportTextOnly: true,          // 支持纯文生视频
        maxImages: 1
    }
];

const dbVideoModels = ref<VideoModelCapability[]>([]);
// const videoModelCapabilities = ref<VideoModelCapability[]>([
//     { id: 'sora-2', name: 'OpenAI Sora', supportSingleImage: true, supportMultipleImages: false, supportFirstLastFrame: false, supportTextOnly: true, maxImages: 1 },
//     { id: 'doubao-seedance-1-5-pro-251215', name: '豆包 (Seedance)', supportSingleImage: true, supportMultipleImages: true, supportFirstLastFrame: true, supportTextOnly: true, maxImages: 6 },
//     { id: 'MiniMax-Hailuo-02', name: '海螺 (MiniMax)', supportSingleImage: true, supportMultipleImages: false, supportFirstLastFrame: true, supportTextOnly: true, maxImages: 2 },
//     { id: 'kling', name: '可灵 (Kling)', supportSingleImage: true, supportMultipleImages: false, supportFirstLastFrame: true, supportTextOnly: true, maxImages: 2 },
//     { id: 'runway', name: 'Runway Gen-3', supportSingleImage: true, supportMultipleImages: false, supportFirstLastFrame: true, supportTextOnly: true, maxImages: 2 },
//     { id: 'pika', name: 'Pika 1.0', supportSingleImage: true, supportMultipleImages: true, supportFirstLastFrame: false, supportTextOnly: true, maxImages: 4 },
//     {
//         id: 'veo-3.1-fast-generate-001',
//         name: 'Google Veo (Vertex AI)',
//         supportSingleImage: true,       // 支持首帧图生视频
//         supportMultipleImages: false,   // 暂不支持多图
//         supportFirstLastFrame: false,   // 暂不支持首尾帧
//         supportTextOnly: true,          // 支持纯文生视频
//         maxImages: 1
//     }
// ]);

// 计算最终显示在下拉框的模型列表
const videoModelCapabilities = computed(() => {
    // 1. 获取数据库配置的模型 ID 集合
    const dbModelIds = new Set(dbVideoModels.value.map(m => m.id));

    // 2. 过滤掉默认列表中已在数据库中存在的，避免重复
    const filteredDefaults = defaultVideoModels.filter(m => !dbModelIds.has(m.id));

    // 3. 数据库优先 + 默认靠后
    return [...dbVideoModels.value, ...filteredDefaults];
});

const loadDbAiConfigs = async () => {
    try {
        const res = await getAiConfigList({
            service_type: 'video',
            is_active: 1,
            pageSize: 100
        });

        if (res.code === 0) {
            const list = res.data?.list || [];
            const extractedModels: VideoModelCapability[] = [];

            list.forEach((config: any) => {
                const models = Array.isArray(config.model) ? config.model : [config.model];
                models.forEach((modelId: string) => {
                    // 如果本地 Map 里有详细能力定义，则取定义，否则给个默认通用定义
                    if (MODEL_CAPABILITY_MAP[modelId]) {
                        extractedModels.push({
                            id: modelId,
                            ...MODEL_CAPABILITY_MAP[modelId],
                            name: `${MODEL_CAPABILITY_MAP[modelId].name} (${config.name})` // 标记来源
                        });
                    } else {
                        extractedModels.push({
                            id: modelId,
                            name: `${modelId} (${config.name})`,
                            supportSingleImage: true,
                            supportMultipleImages: false,
                            supportFirstLastFrame: false,
                            supportTextOnly: true,
                            maxImages: 1
                        });
                    }
                });
            });
            dbVideoModels.value = extractedModels;

            // 如果当前选中的模型不在新列表中，默认选中第一个
            if (dbVideoModels.value.length > 0 && !videoModelCapabilities.value.find(m => m.id === selectedVideoModel.value)) {
                selectedVideoModel.value = dbVideoModels.value[0].id;
            }
        }
    } catch (e) {
        console.error("加载 AI 配置失败:", e);
    }
};


const currentModelCapability = computed(() => {
    return videoModelCapabilities.value.find(m => m.id === selectedVideoModel.value);
});

const availableReferenceModes = computed(() => {
    const capability = currentModelCapability.value;
    if (!capability) return [];
    const modes: Array<{ value: string; label: string; description?: string }> = [];
    if (capability.supportTextOnly) modes.push({ value: "none", label: "纯文本", description: "不使用参考图" });
    if (capability.supportSingleImage) modes.push({ value: "single", label: "单图参考", description: "使用单张参考图" });
    if (capability.supportFirstLastFrame) modes.push({ value: "first_last", label: "首尾帧", description: "使用首帧和尾帧" });
    if (capability.supportMultipleImages) modes.push({ value: "multiple", label: "多图", description: `最多${capability.maxImages}张` });
    return modes;
});

watch(selectedVideoModel, () => {
    selectedImagesForVideo.value = [];
    selectedLastImageForVideo.value = null;
    referenceMode.value = availableReferenceModes.value[0]?.value || 'none';
});

watch(referenceMode, () => {
    selectedImagesForVideo.value = [];
    selectedLastImageForVideo.value = null;
});

const currentStoryboard = computed(() => storyboards.value.find(s => String(s.id) === String(currentStoryboardId.value)))

// 兼容旧分镜：历史数据把 Result 存在 visualDesc 中，编辑时同步展示但不丢失原字段。
watch(currentStoryboard, (shot) => {
    if (shot && !shot.result && shot.visualDesc) shot.result = shot.visualDesc
}, { immediate: true })

const previousStoryboard = computed(() => {
    if (!currentStoryboardId.value || storyboards.value.length < 2) return null;
    const currentIndex = storyboards.value.findIndex((s) => String(s.id) === String(currentStoryboardId.value));
    if (currentIndex <= 0) return null;
    return storyboards.value[currentIndex - 1];
});

const previousStoryboardLastFrames = computed(() => {
    if (!previousStoryboard.value || !previousStoryboard.value.frameImages) return [];
    return previousStoryboard.value.frameImages.filter((img: any) => img.frameType === 'last' && (!img.imageType || img.imageType === 'shot'));
});

const currentFrameImagesFiltered = (type: string) => {
    if (!currentStoryboard.value || !currentStoryboard.value.frameImages) return [];
    return currentStoryboard.value.frameImages.filter((img: any) =>
        img.frameType === type &&
        (!img.imageType || img.imageType === 'shot' || img.imageType === 'reference')
    );
};

const getAllAvailableImages = () => {
    const curr = currentStoryboard.value?.frameImages || [];
    const prev = previousStoryboardLastFrames.value || [];
    return [...curr, ...prev];
};

const isImageSelected = (img: any) => {
    if (referenceMode.value === 'first_last' && selectedVideoFrameType.value === 'last') {
        return selectedLastImageForVideo.value === img.id;
    }
    return selectedImagesForVideo.value.includes(img.id);
}

const isPreviousFrameSelected = (prevImg: any) => {
    if (referenceMode.value === 'single' || referenceMode.value === 'first_last') {
        return selectedImagesForVideo.value[0] === prevImg.id;
    } else if (referenceMode.value === 'multiple') {
        return selectedImagesForVideo.value.includes(prevImg.id);
    }
    return false;
}

const handlePreviousImageSelect = (img: any) => {
    if (!referenceMode.value || referenceMode.value === 'none') {
        MessagePlugin.warning("请先选择参考图模式");
        return;
    }

    if (referenceMode.value === 'single' || referenceMode.value === 'first_last') {
        if (selectedImagesForVideo.value[0] === img.id) {
            selectedImagesForVideo.value = [];
        } else {
            selectedImagesForVideo.value = [img.id];
        }
    } else if (referenceMode.value === 'multiple') {
        const index = selectedImagesForVideo.value.indexOf(img.id);
        if (index > -1) {
            selectedImagesForVideo.value.splice(index, 1);
        } else {
            if (selectedImagesForVideo.value.length >= (currentModelCapability.value?.maxImages || 6)) {
                MessagePlugin.warning(`最多只能选择${currentModelCapability.value?.maxImages}张图片`);
                return;
            }
            selectedImagesForVideo.value.push(img.id);
        }
    }
};

const handleImageSelect = (img: any) => {
    if (!referenceMode.value || referenceMode.value === 'none') {
        MessagePlugin.warning("请先选择参考图模式");
        return;
    }

    const imageId = img.id;
    const isClickingLastFrame = selectedVideoFrameType.value === 'last';

    if (referenceMode.value === 'multiple') {
        const index = selectedImagesForVideo.value.indexOf(imageId);
        if (index > -1) {
            selectedImagesForVideo.value.splice(index, 1);
        } else {
            if (selectedImagesForVideo.value.length >= (currentModelCapability.value?.maxImages || 6)) {
                MessagePlugin.warning(`最多只能选择${currentModelCapability.value?.maxImages}张图片`);
                return;
            }
            selectedImagesForVideo.value.push(imageId);
        }
        return;
    }

    if (referenceMode.value === 'single') {
        if (selectedImagesForVideo.value[0] === imageId) {
            selectedImagesForVideo.value = [];
        } else {
            selectedImagesForVideo.value = [imageId];
        }
        return;
    }

    if (referenceMode.value === 'first_last') {
        if (isClickingLastFrame) {
            if (selectedLastImageForVideo.value === imageId) {
                selectedLastImageForVideo.value = null;
            } else {
                selectedLastImageForVideo.value = imageId;
            }
        } else {
            if (selectedImagesForVideo.value[0] === imageId) {
                selectedImagesForVideo.value = [];
            } else {
                selectedImagesForVideo.value = [imageId];
            }
        }
    }
};

const getFrameTypeName = (type: string) => {
    const map: Record<string, string> = { first: '首帧', last: '尾帧', action: '动作序列', key: '关键帧' };
    return map[type] || type;
}

const removeSelectedImage = (imageId: number) => {
    if (selectedLastImageForVideo.value === imageId) {
        selectedLastImageForVideo.value = null;
        return;
    }
    const index = selectedImagesForVideo.value.indexOf(imageId);
    if (index > -1) {
        selectedImagesForVideo.value.splice(index, 1);
    }
};

const singleRefImage = computed(() => {
    return getAllAvailableImages().find(i => i.id === selectedImagesForVideo.value[0]);
})
const firstRefImage = computed(() => {
    return getAllAvailableImages().find(i => i.id === selectedImagesForVideo.value[0]);
})
const lastRefImage = computed(() => {
    return getAllAvailableImages().find(i => i.id === selectedLastImageForVideo.value);
})
const getMultiRefImage = (index: number) => {
    return getAllAvailableImages().find(i => i.id === selectedImagesForVideo.value[index - 1]);
}


const currentScene = computed(() => {
    if (!currentStoryboard.value || !currentStoryboard.value.sceneId) return null
    return sceneList.value.find(s => s.id === currentStoryboard.value.sceneId)
})

const selectedCharacters = computed(() => {
    if (!currentStoryboard.value?.characters) return []
    return currentStoryboard.value.characters.map((c: any) => typeof c === 'object' ? c.id : c)
})

const selectedProps = computed(() => {
    if (!currentStoryboard.value?.props) return []
    return currentStoryboard.value.props.map((p: any) => typeof p === 'object' ? p.id : p)
})

const currentFramePromptText = computed({
    get() {
        if (!currentStoryboard.value) return '';
        const prompts = currentStoryboard.value.framePrompts || [];
        const fp = prompts.find((p: any) => p.frameType === selectedFrameType.value);
        if (!fp && selectedFrameType.value === 'first' && currentStoryboard.value.imagePrompt) {
            return currentStoryboard.value.imagePrompt;
        }
        return fp ? fp.prompt : '';
    },
    set(val) {
        if (!currentStoryboard.value) return;
        if (!currentStoryboard.value.framePrompts) {
            currentStoryboard.value.framePrompts = [];
        }
        const fpIndex = currentStoryboard.value.framePrompts.findIndex((p: any) => p.frameType === selectedFrameType.value);
        if (fpIndex > -1) {
            currentStoryboard.value.framePrompts[fpIndex].prompt = val;
        } else {
            currentStoryboard.value.framePrompts.push({
                frameType: selectedFrameType.value,
                prompt: val
            });
        }
        if (selectedFrameType.value === 'first') {
            currentStoryboard.value.imagePrompt = val;
        }
    }
})

const currentFrameImages = computed(() => {
    if (!currentStoryboard.value || !currentStoryboard.value.frameImages) return [];
    return currentStoryboard.value.frameImages.filter((img: any) =>
        img.frameType === selectedFrameType.value &&
        (!img.imageType || img.imageType === 'shot')
    );
})

const getCharacterById = (id: number) => availableCharacters.value.find(c => c.id === id)
const getPropById = (id: number) => availableProps.value.find(p => p.id === id)

const getAuthToken = () => localStorage.getItem('token')

const uploadConfig = reactive({
    action: import.meta.env.VITE_API_URL + '/admin/v1/upload/singleUpload',
    headers: computed(() => ({ 'Authorization': `${getAuthToken()}` })),
    sizeLimit: 5 * 1024 * 1024,
    allowedFormats: ['image/jpeg', 'image/jpg', 'image/png', 'image/gif', 'image/webp']
})

const syncGeneratedVideos = () => {
    generatedVideos.value = [];
    if (currentStoryboard.value) {
        if (currentStoryboard.value.generateVideos && currentStoryboard.value.generateVideos.length > 0) {
            generatedVideos.value = currentStoryboard.value.generateVideos.map((v: any) => ({
                id: v.id,
                status: 2,
                url: v.videoUrl || v.video_url || v.url
            }));
        }
        else if (currentStoryboard.value.videoUrl || currentStoryboard.value.video_url) {
            generatedVideos.value.push({
                id: 'main-' + currentStoryboard.value.id,
                status: 2,
                url: currentStoryboard.value.videoUrl || currentStoryboard.value.video_url
            });
        }
    }
};

const initData = async () => {
    loading.value = true
    try {
        const res = await findProjects(dramaId)
        if (res.code === 0) project.value = res.data

        const sceneRes = await getScenesList({ projectId: dramaId, pageSize: 100 })
        if (sceneRes.code === 0) sceneList.value = sceneRes.data?.list || []

        const charRes = await getCharactersList({ projectId: dramaId, pageSize: 100 })
        if (charRes.code === 0) availableCharacters.value = charRes.data?.list || []

        const propRes = await getPropsList({ projectId: dramaId, pageSize: 100 })
        if (propRes.code === 0) availableProps.value = propRes.data?.list || []
        await loadDbAiConfigs();
        await loadShotsData()
        await loadVideoAssets()
        await loadMergeList()

    } catch (e) { console.error(e) } finally { loading.value = false }
}

const loadShotsData = async () => {
    const scriptRes = await getScriptsList({ projectId: dramaId, page: 1, pageSize: 100 })
    const list = scriptRes.data?.list || []
    const targetScript = list.find((s: any) => Number(s.episodeNo) === episodeNumber)

    if (targetScript) {
        currentScriptId.value = targetScript.id
        const shotRes = await getShotsList({ scriptId: targetScript.id, pageSize: 1000 })
        if (shotRes.code === 0) {
            storyboards.value = shotRes.data?.list || shotRes.data || []
            if (storyboards.value.length > 0 && !currentStoryboardId.value) {
                selectStoryboard(storyboards.value[0].id)
            } else {
                syncGeneratedVideos()
            }
        }
    }
}

const loadVideoAssets = async () => {
    if (!currentScriptId.value) return;
    try {
        const res = await request.get({
            url: '/source',
            params: { scriptId: currentScriptId.value, pageSize: 100 }
        });
        if (res.code === 0 || res.success) {
            videoAssets.value = res.data?.list || res.data || [];
        }
    } catch (e) {
        console.error("加载素材库失败:", e);
    }
}

// 🔴 加载合并任务列表
const loadMergeList = async () => {
    if (!currentScriptId.value) return;
    try {
        const res = await request.get({
            url: '/shot_video_merges', // 这里调用正确的接口加载合并列表
            params: { scriptId: currentScriptId.value, pageSize: 100 }
        });
        if (res.code === 0 || res.success) {
            videoMerges.value = res.data?.list || res.data || [];
        }
    } catch (e) {
        console.error("加载合并任务失败:", e);
    }
}

const toggleCharacterInShot = async (charId: number) => {
    if (!currentStoryboard.value) return
    let chars = currentStoryboard.value.characters || []
    const idx = chars.findIndex((c: any) => (typeof c === 'object' ? c.id === charId : c === charId))

    if (idx > -1) {
        chars.splice(idx, 1)
    } else {
        const fullChar = availableCharacters.value.find(c => c.id === charId)
        if (fullChar) chars.push(fullChar)
    }
    currentStoryboard.value.characters = chars
    currentStoryboard.value.characterIds = chars.map((c: any) => typeof c === 'object' ? c.id : c)
    await saveStoryboardField()
}

const togglePropInShot = async (propId: number) => {
    if (!currentStoryboard.value) return
    let propsArray = currentStoryboard.value.props || []
    const idx = propsArray.findIndex((p: any) => (typeof p === 'object' ? p.id === propId : p === propId))

    if (idx > -1) {
        propsArray.splice(idx, 1)
    } else {
        const fullProp = availableProps.value.find(p => p.id === propId)
        if (fullProp) propsArray.push(fullProp)
    }
    currentStoryboard.value.props = propsArray
    currentStoryboard.value.propIds = propsArray.map((p: any) => typeof p === 'object' ? p.id : p)
    await saveStoryboardField()
}

const handleDragStart = (e: DragEvent, item: any, type: 'storyboard' | 'asset') => {
    if (e.dataTransfer) {
        const videoUrl = type === 'asset' ? item.videoUrl || item.url : item.videoUrl
        const payload = {
            id: item.id,
            name: item.title || item.name,
            url: videoUrl,
            duration: item.duration || 5,
            type: 'video',
            shotId: type === 'storyboard' ? item.id : item.shotId
        }
        e.dataTransfer.setData('application/json', JSON.stringify(payload))
        e.dataTransfer.effectAllowed = 'copy'
    }
}

const handleTimelineDrop = (data: any) => {
    if (!data || !data.item) return;
    const payload = data.item;

    let insertStart = data.start;
    if (insertStart === undefined || insertStart < 0) {
        const trackClips = timelineClips.value.filter(c => c.track === (data.track || 0));
        insertStart = trackClips.length > 0 ? Math.max(...trackClips.map(c => c.start + c.duration)) : 0;
    }

    const newClip = {
        id: 'clip_' + Date.now() + '_' + Math.floor(Math.random() * 1000),
        assetId: payload.id,
        shotId: payload.shotId || currentStoryboardId.value,
        url: payload.url,
        cover: getImageUrl(payload.url),
        start: insertStart,
        duration: payload.duration || 5,
        track: data.track || 0,
        type: payload.type || 'video',
        transition: { type: 'none', duration: 0.5 }
    };

    timelineClips.value.push(newClip);
    MessagePlugin.success('已加入时间线');
}

const addAssetToTimeline = (asset: any) => {
    const trackClips = timelineClips.value.filter(c => c.track === 0);
    const maxEndTime = trackClips.length > 0 ? Math.max(...trackClips.map(c => c.start + c.duration)) : 0;
    const videoUrl = asset.videoUrl || asset.video_url || asset.url;

    timelineClips.value.push({
        id: 'clip_' + Date.now() + '_' + Math.floor(Math.random() * 1000),
        assetId: asset.id,
        shotId: asset.shotId || asset.shot_id || currentStoryboardId.value,
        url: getImageUrl(videoUrl),
        cover: getImageUrl(videoUrl),
        start: maxEndTime,
        duration: asset.duration || 5,
        track: 0,
        type: 'video',
        transition: { type: 'none', duration: 0.5 }
    });

    MessagePlugin.success('已添加到主视频轨道');
}

// 🔴 一键将素材库的所有素材添加到时间线
const addAllAssetsToTimeline = () => {
    if (videoAssets.value.length === 0) {
        MessagePlugin.warning('素材库为空');
        return;
    }

    // 获取当前时间线主轨道的最后时间点，以便追加
    const trackClips = timelineClips.value.filter(c => c.track === 0);
    let currentStartTime = trackClips.length > 0 ? Math.max(...trackClips.map(c => c.start + c.duration)) : 0;

    // 按分镜序号排序素材
    const sortedAssets = [...videoAssets.value].sort((a: any, b: any) => {
        const numA = a.shotNumber || a.shot_number || 0;
        const numB = b.shotNumber || b.shot_number || 0;
        return numA - numB;
    });

    sortedAssets.forEach((asset: any, index: number) => {
        const clipDuration = asset.duration || 5;
        const videoUrl = asset.videoUrl || asset.video_url || asset.url;
        timelineClips.value.push({
            id: 'clip_auto_' + index + '_' + Date.now(),
            assetId: asset.id,
            shotId: asset.shotId || asset.shot_id || 0,
            url: getImageUrl(videoUrl),
            cover: getImageUrl(videoUrl), // 🔴 添加封面
            start: currentStartTime,
            duration: clipDuration,
            track: 0,
            type: 'video',
            transition: { type: 'none', duration: 0.5 }
        });
        currentStartTime += clipDuration;
    });

    MessagePlugin.success('已将所有素材追加到时间线');
}

const goBack = () => router.back()
const loadData = () => { initData(); MessagePlugin.success('数据已刷新') }

const selectStoryboard = (id: number | string) => {
    currentStoryboardId.value = id
    syncGeneratedVideos()
}

const handleTimelineSelect = async (clip: any) => {
    const targetClip = timelineClips.value.find(c => c.id === clip.id);
    if (targetClip) {
        selectedTimelineClip.value = targetClip;
        activeTab.value = 'clip';
        if (targetClip.shotId) {
            selectStoryboard(targetClip.shotId);
        }
    } else {
        selectedTimelineClip.value = null;
    }

    currentPreviewUrl.value = clip.url;
    await nextTick();
    if (mainPlayerRef.value) {
        mainPlayerRef.value.currentTime = 0;
        mainPlayerRef.value.play().catch(() => { });
    }
}

const removeClipFromTimeline = (clipId: string) => {
    const idx = timelineClips.value.findIndex(c => c.id === clipId)
    if (idx > -1) {
        timelineClips.value.splice(idx, 1)
        if (selectedTimelineClip.value && selectedTimelineClip.value.id === clipId) {
            selectedTimelineClip.value = null;
            if (activeTab.value === 'clip') activeTab.value = 'shot';
        }
    }
}

const updateCurrentTime = (time: number) => {
    currentTime.value = time
    const activeClip = timelineClips.value.find(c => time >= c.start && time < c.start + c.duration)
    if (activeClip && activeClip.url) {
        if (currentPreviewUrl.value !== activeClip.url) currentPreviewUrl.value = activeClip.url
        const offset = time - activeClip.start
        if (mainPlayerRef.value && Math.abs(mainPlayerRef.value.currentTime - offset) > 0.5) {
            mainPlayerRef.value.currentTime = offset
        }
    }
}

const handleAddStoryboard = async () => {
    const newShot = {
        projectId: Number(dramaId),
        scriptId: currentScriptId.value,
        title: `新镜头 ${storyboards.value.length + 1}`,
        durationMs: 3000,
        shotType: '中景',
        angle: '平视',
        cameraMovement: '固定'
    }
    try {
        MessagePlugin.success('添加成功 (Mock)')
        storyboards.value.push({ id: Date.now(), ...newShot })
    } catch { MessagePlugin.error('添加失败') }
}

const handleDeleteStoryboard = async (shot: any) => { MessagePlugin.success('删除成功') }

const linkSceneToShot = async (scene: any) => {
    if (!currentStoryboard.value) return
    currentStoryboard.value.sceneId = scene.id
    await saveStoryboardField()
    showSceneSelector.value = false; MessagePlugin.success('已关联场景')
}

const saveStoryboardField = async () => {
    if (!currentStoryboard.value) return
    saving.value = true
    try {
        const payload = {
            ...currentStoryboard.value,
            characterIds: currentStoryboard.value.characters?.map((c: any) => typeof c === 'object' ? c.id : c) || [],
            propIds: currentStoryboard.value.props?.map((p: any) => typeof p === 'object' ? p.id : p) || []
        }
        delete payload.characters
        delete payload.props
        delete payload.scenes
        delete payload.frameImages
        delete payload.framePrompts

        await updateShots(payload.id, payload)
    } catch {
        MessagePlugin.error('保存失败')
    } finally {
        saving.value = false
    }
}

const updateShotDurationMs = (secVal: number) => {
    if (!currentStoryboard.value) return
    currentStoryboard.value.durationMs = secVal * 1000
    saveStoryboardField()
}

const extractFramePrompt = async () => {
    if (!currentStoryboard.value) return;
    extractingPrompt.value = true;
    try {
        const res = await extractFramePromptTask({
            shotId: currentStoryboard.value.id,
            frameType: selectedFrameType.value
        });
        const taskId = res.data?.task_id || res.data?.taskId || res.data?.data?.task_id;

        if (taskId) {
            MessagePlugin.loading('AI 正在提取提示词...');
            const timer = setInterval(async () => {
                try {
                    const taskRes = await findTasks(taskId);
                    const taskData = taskRes.data?.data || taskRes.data;
                    const status = taskData?.status;
                    if (status === 'completed' || status === 2) {
                        clearInterval(timer);
                        extractingPrompt.value = false;
                        MessagePlugin.success('提示词提取成功');
                        await loadShotsData();
                    } else if (status === 'failed' || status === 3) {
                        clearInterval(timer);
                        extractingPrompt.value = false;
                        MessagePlugin.error(taskData?.error || '提取失败');
                    }
                } catch (e) {
                    clearInterval(timer);
                    extractingPrompt.value = false;
                }
            }, 2000);
        } else {
            extractingPrompt.value = false;
            MessagePlugin.error('任务提交失败');
        }
    } catch (e) {
        extractingPrompt.value = false;
        MessagePlugin.error('请求异常');
    }
}

const generateFrameImage = async () => {
    if (!currentStoryboard.value) return;
    if (!currentFramePromptText.value) {
        MessagePlugin.warning('请先输入或提取提示词');
        return;
    }

    generatingImage.value = true;
    try {
        const res = await generateImageByPromptTask({
            shotId: currentStoryboard.value.id,
            frameType: selectedFrameType.value,
            prompt: currentFramePromptText.value
        });

        const taskId = res.data?.task_id || res.data?.taskId || res.data?.data?.task_id;

        if (taskId) {
            MessagePlugin.loading('AI 正在生成画面，请耐心等待...');
            const timer = setInterval(async () => {
                try {
                    const taskRes = await findTasks(taskId);
                    const taskData = taskRes.data?.data || taskRes.data;
                    const status = taskData?.status;

                    if (status === 'completed' || status === 2) {
                        clearInterval(timer);
                        generatingImage.value = false;
                        MessagePlugin.success('画面生成成功！');
                        await loadShotsData();
                    } else if (status === 'failed' || status === 3) {
                        clearInterval(timer);
                        generatingImage.value = false;
                        MessagePlugin.error(taskData?.error || '生成失败');
                    }
                } catch (e) {
                    clearInterval(timer);
                    generatingImage.value = false;
                }
            }, 3000);
        } else {
            generatingImage.value = false;
            MessagePlugin.error('生图任务提交失败');
        }
    } catch (e) {
        generatingImage.value = false;
        MessagePlugin.error('生图请求异常');
    }
}

const beforeUpload = (file: any) => {
    if (!uploadConfig.allowedFormats.includes(file.type)) {
        MessagePlugin.error('不支持的文件格式')
        return false
    }
    if (file.size > uploadConfig.sizeLimit) {
        MessagePlugin.error('图片大小不能超过 5MB')
        return false
    }
    uploadingImage.value = true
    return true
}

const handleUploadFail = () => {
    uploadingImage.value = false
    MessagePlugin.error('上传失败')
}

const handleUploadImageSuccess = async (ctx: any) => {
    uploadingImage.value = false
    const response = ctx.response

    if (response?.code === 0 || response?.code === 200) {
        let fileUrl = response.data.file_url || response.data.url
        if (fileUrl && fileUrl.startsWith('/')) {
            fileUrl = import.meta.env.VITE_API_URL.replace(/\/admin\/v1$/, '').replace(/\/v1$/, '') + fileUrl
        }

        if (currentStoryboard.value) {
            try {
                const res = await createShotFrameImages({
                    projectId: Number(dramaId),
                    shotId: currentStoryboard.value.id,
                    frameType: selectedFrameType.value,
                    imageType: 'shot',
                    imageUrl: fileUrl
                });

                if (res.code === 0) {
                    MessagePlugin.success('图片添加成功');
                    if (!currentStoryboard.value.frameImages) {
                        currentStoryboard.value.frameImages = [];
                    }
                    currentStoryboard.value.frameImages.unshift(res.data);

                    if (!currentStoryboard.value.imageUrl || selectedFrameType.value === 'first') {
                        currentStoryboard.value.imageUrl = fileUrl;
                        saveStoryboardField();
                    }
                } else {
                    MessagePlugin.error(res.message || '图片数据保存失败');
                }
            } catch (err) {
                console.error(err);
                MessagePlugin.error('图片数据请求异常');
            }
        }
    } else {
        MessagePlugin.error(response?.msg || '上传失败')
    }
}

const handleUploadRefSuccess = async (ctx: any, targetSlot: string) => {
    const response = ctx.response;
    if (response?.code === 0 || response?.code === 200) {
        let fileUrl = response.data.file_url || response.data.url;
        if (fileUrl && fileUrl.startsWith('/')) {
            fileUrl = import.meta.env.VITE_API_URL.replace(/\/admin\/v1$/, '').replace(/\/v1$/, '') + fileUrl;
        }
        if (currentStoryboard.value) {
            try {
                const res = await createShotFrameImages({
                    projectId: Number(dramaId),
                    shotId: currentStoryboard.value.id,
                    frameType: targetSlot,
                    imageType: 'reference',
                    imageUrl: fileUrl
                });

                if (res.code === 0) {
                    MessagePlugin.success('上传并设置成功');
                    if (!currentStoryboard.value.frameImages) {
                        currentStoryboard.value.frameImages = [];
                    }
                    currentStoryboard.value.frameImages.unshift(res.data);

                    if (targetSlot === 'single' || targetSlot === 'first') {
                        selectedImagesForVideo.value = [res.data.id];
                    } else if (targetSlot === 'last') {
                        selectedLastImageForVideo.value = res.data.id;
                    } else if (targetSlot.startsWith('multi')) {
                        if (selectedImagesForVideo.value.length < (currentModelCapability.value?.maxImages || 6)) {
                            selectedImagesForVideo.value.push(res.data.id);
                        }
                    }
                }
            } catch (err) {
                MessagePlugin.error('请求异常');
            }
        }
    } else {
        MessagePlugin.error(response?.msg || '上传失败');
    }
};

const deleteImage = async (img: any) => {
    if (!img.id || !currentStoryboard.value) return;

    const confirmDialog = DialogPlugin.confirm({
        header: '确认删除',
        body: '确定要删除这张图片吗？',
        onConfirm: async () => {
            try {
                const res = await deleteShotFrameImages(img.id);
                if (res.code === 0 || res.code === 200) {
                    const idx = currentStoryboard.value.frameImages.findIndex((i: any) => i.id === img.id);
                    if (idx > -1) {
                        currentStoryboard.value.frameImages.splice(idx, 1);
                    }
                    removeSelectedImage(img.id);
                    MessagePlugin.success('删除成功');
                } else {
                    MessagePlugin.error(res.message || '删除失败');
                }
            } catch (e) {
                MessagePlugin.error('删除请求异常');
            } finally {
                confirmDialog.destroy();
            }
        },
        onCancel: () => {
            confirmDialog.destroy();
        }
    });
}

const openCropDialog = (img: any) => {
    let fullUrl = img.url || img.imageUrl;
    if (!fullUrl.startsWith('http')) {
        fullUrl = getImageUrl(fullUrl);
    }
    cropImageUrl.value = fullUrl;
    showCropDialog.value = true;
}

const handleCropSave = (newUrl: string) => { showCropDialog.value = false }

const handleGridSuccess = async (data: { url: string, frameType: string }) => {
    if (data && data.url && currentStoryboard.value) {
        try {
            const res = await createShotFrameImages({
                projectId: Number(dramaId),
                shotId: currentStoryboard.value.id,
                frameType: data.frameType,
                imageType: 'shot',
                imageUrl: data.url
            });

            if (res.code === 0) {
                if (!currentStoryboard.value.frameImages) {
                    currentStoryboard.value.frameImages = [];
                }
                currentStoryboard.value.frameImages.unshift(res.data);
                MessagePlugin.success('宫格图保存成功');
            } else {
                MessagePlugin.error(res.message || '宫格图保存记录失败');
            }
        } catch (e) {
            console.error(e);
            MessagePlugin.error('宫格图保存异常');
        }
    }
}

const getStatusText = (status: string | number) => {
    if (status === 'completed' || status === 2) return '生成成功'
    if (status === 'processing' || status === 1) return '生成中'
    if (status === 'failed' || status === 3) return '生成失败'
    return '等待中'
}

const generateVideo = async () => {
    if (!selectedVideoModel.value) {
        MessagePlugin.warning("请先选择视频生成模型");
        return;
    }
    if (!currentStoryboard.value) return;

    if (referenceMode.value === 'single' && selectedImagesForVideo.value.length === 0) {
        MessagePlugin.warning("请上传或选择单图参考");
        return;
    }
    if (referenceMode.value === 'first_last' && (selectedImagesForVideo.value.length === 0 || !selectedLastImageForVideo.value)) {
        MessagePlugin.warning("请选择完整的首尾帧参考图");
        return;
    }

    generatingVideo.value = true;
    try {
        const requestPayload: any = {
            projectId: Number(dramaId),
            shotId: currentStoryboard.value.id,
            model: selectedVideoModel.value,
            duration: currentShotDurationSeconds.value,
            prompt: currentStoryboard.value.videoPrompt || currentStoryboard.value.imagePrompt || currentStoryboard.value.visualDesc || '',
            referenceMode: referenceMode.value
        }

        const allImages = getAllAvailableImages();

        if (referenceMode.value === 'single') {
            const img = allImages.find(i => i.id === selectedImagesForVideo.value[0]);
            requestPayload.imageUrl = img?.url || img?.imageUrl;
        } else if (referenceMode.value === 'first_last') {
            const firstImg = allImages.find(i => i.id === selectedImagesForVideo.value[0]);
            const lastImg = allImages.find(i => i.id === selectedLastImageForVideo.value);
            requestPayload.firstFrameUrl = firstImg?.url || firstImg?.imageUrl;
            requestPayload.lastFrameUrl = lastImg?.url || lastImg?.imageUrl;
        } else if (referenceMode.value === 'multiple') {
            requestPayload.imageUrls = selectedImagesForVideo.value.map(id => {
                const img = allImages.find(i => i.id === id);
                return img?.url || img?.imageUrl;
            });
        }

        const res = await generateVideoTask(requestPayload);
        const taskId = res.data?.task_id || res.data?.taskId || res.data?.data?.task_id;

        if (taskId) {
            MessagePlugin.loading('视频任务已提交，正在生成，请耐心等待...');

            const newVideoRecord = {
                id: Date.now(),
                status: 'processing',
                taskId: taskId
            };
            generatedVideos.value.unshift(newVideoRecord);

            const timer = setInterval(async () => {
                try {
                    const taskRes = await findTasks(taskId);
                    const payload = taskRes.data?.data || taskRes.data || taskRes;
                    const taskData = payload?.task || payload;
                    const status = taskData?.status;
                    const statusName = taskData?.statusName || taskData?.status_name;

                    const idx = generatedVideos.value.findIndex(v => v.taskId === taskId);

                    if (status === 'completed' || status === 'succeeded' || statusName === 'succeeded' || status === 2 || taskData?.process >= 100) {
                        clearInterval(timer);
                        generatingVideo.value = false;
                        MessagePlugin.success('视频生成成功！');

                        if (idx > -1) {
                            let resultData = taskData.result;
                            if (typeof resultData === 'string' && resultData.trim() !== '') {
                                try { resultData = JSON.parse(resultData); } catch (e) { }
                            }
                            generatedVideos.value[idx].status = 2;
                            generatedVideos.value[idx].url = resultData?.url || resultData?.video_url || resultData?.videoUrl || taskData?.videoUrl || taskData?.video_url;

                            if (currentStoryboard.value) {
                                currentStoryboard.value.videoUrl = generatedVideos.value[idx].url;
                            }
                        }
                        // Worker 完成后重新读取镜头，拿到数据库中最终保存的视频地址。
                        await loadShotsData();
                        syncGeneratedVideos();
                        const persisted = generatedVideos.value.find(v => v.taskId === taskId);
                        if (persisted?.url) currentPreviewUrl.value = getImageUrl(persisted.url);
                    } else if (status === 'failed' || statusName === 'failed' || status === 3 || status === 'cancelled' || statusName === 'cancelled' || status === 4) {
                        clearInterval(timer);
                        generatingVideo.value = false;
                        MessagePlugin.error(taskData?.errorMsg || taskData?.error_msg || taskData?.error || '生成失败');
                        if (idx > -1) generatedVideos.value[idx].status = 3;
                    }
                } catch (e) {
                    clearInterval(timer);
                    generatingVideo.value = false;
                }
            }, 5000);
        } else {
            generatingVideo.value = false;
            MessagePlugin.error('视频任务提交失败');
        }
    } catch (e) {
        generatingVideo.value = false;
        MessagePlugin.error("任务请求异常");
    }
}

const playVideo = async (video: any) => {
    const url = video.url || video.videoUrl || video.video_url;
    if (url) {
        currentPreviewUrl.value = getImageUrl(url);
        await nextTick();
        if (mainPlayerRef.value) {
            mainPlayerRef.value.currentTime = 0;
            mainPlayerRef.value.play().catch(() => { });
        }
    }
}

const deleteMergeItem = async (mergeId: number) => {
    const confirm = DialogPlugin.confirm({
        header: '删除合并记录',
        body: '确定要删除这条合成记录吗？',
        onConfirm: async () => {
            try {
                const res = await request.delete({ url: `/shot_video_merges/${mergeId}` });
                if (res.code === 0 || res.success) {
                    MessagePlugin.success('删除成功');
                    loadMergeList();
                } else {
                    MessagePlugin.error(res.message || '删除失败');
                }
            } catch (e) {
                MessagePlugin.error('请求异常');
            } finally {
                confirm.destroy();
            }
        },
        onCancel: () => confirm.destroy()
    });
}

// 🔴 ：合并视频接口对接
const exportVideo = async () => {
    if (!timelineClips.value || timelineClips.value.length === 0) {
        MessagePlugin.warning("时间线上没有视频片段，无法合成");
        return;
    }

    mergingVideo.value = true;
    try {
        const formattedClips = timelineClips.value.map((clip, index) => ({
            assetId: clip.assetId || null,
            shotId: clip.storyboardId || clip.shotId,
            order: index + 1, // 增加顺序字段
            startTime: clip.start,
            endTime: clip.start + clip.duration,
            duration: clip.duration,
            url: clip.url,
            transition: clip.transition || { type: "none" }
        }));

        const requestPayload = {
            projectId: Number(dramaId),
            episodeNumber: episodeNumber,
            scriptId: Number(currentScriptId.value),
            clips: formattedClips
        };

        const res = await mergeVideoTask(requestPayload);
        const responseData = res?.data?.data || res?.data || res;
        const taskId = responseData?.task_id || responseData?.taskId || responseData?.id;

        if (taskId) {
            MessagePlugin.loading('合成任务已提交，系统正在拼命合成中...');
            loadMergeList(); // 刷新列表以展示 pending 状态
            activeTab.value = 'merge';

            const timer = setInterval(async () => {
                try {
                    const taskRes = await findTasks(taskId);
                    const payload = taskRes.data?.data || taskRes.data || taskRes;
                    const taskData = payload?.task || payload;
                    const status = taskData?.status;
                    const statusName = taskData?.statusName || taskData?.status_name;

                    if (status === 'completed' || status === 'succeeded' || statusName === 'succeeded' || status === 2 || taskData?.process >= 100) {
                        clearInterval(timer);
                        mergingVideo.value = false;
                        MessagePlugin.success('视频合成完成！');
                        loadMergeList(); // 刷新列表展示最新结果
                    } else if (status === 'failed' || statusName === 'failed' || status === 3 || status === 'cancelled' || statusName === 'cancelled' || status === 4) {
                        clearInterval(timer);
                        mergingVideo.value = false;
                        MessagePlugin.error(taskData?.errorMsg || taskData?.error_msg || taskData?.error || '合成失败');
                        loadMergeList();
                    }
                } catch (e) {
                    clearInterval(timer);
                    mergingVideo.value = false;
                }
            }, 3000);
        } else {
            mergingVideo.value = false;
            MessagePlugin.error('任务提交失败');
        }
    } catch (e) {
        mergingVideo.value = false;
        MessagePlugin.error("任务请求异常");
    }
}

// 单个添加到素材库
const addVideoToAssets = async (video: any) => {
    if (!currentStoryboard.value) return;
    try {
        const payload = {
            projectId: Number(dramaId),
            scriptId: Number(currentScriptId.value),
            shotId: Number(currentStoryboard.value.id),
            shotNumber: Number(currentStoryboard.value.sequenceNo),
            videoUrl: video.url || video.videoUrl || video.video_url
        };
        const res = await createSource(payload as any);
        if (res.code === 0 || res.success) {
            MessagePlugin.success('已添加到素材库');
            loadVideoAssets();
        } else {
            MessagePlugin.error(res.message || res.msg || '添加素材失败');
        }
    } catch (e) {
        console.error(e);
        MessagePlugin.error('请求异常');
    }
}

// 添加全部到素材库
const addAllVideosToAssets = async () => {
    const completedVideos = generatedVideos.value.filter(v => v.status === 2 || v.status === 'completed' || v.status === 'succeeded');
    if (completedVideos.length === 0) {
        MessagePlugin.warning('没有可添加的已完成视频');
        return;
    }

    let successCount = 0;
    MessagePlugin.loading('正在批量添加...');

    for (const video of completedVideos) {
        try {
            const payload = {
                projectId: Number(dramaId),
                scriptId: Number(currentScriptId.value),
                shotId: Number(currentStoryboard.value?.id || video.shot_id || 0),
                shotNumber: Number(currentStoryboard.value?.sequenceNo || 0),
                videoUrl: video.url || video.videoUrl || video.video_url
            };
            const res = await createSource(payload as any);
            if (res.code === 0 || res.success) successCount++;
        } catch (e) { }
    }

    MessagePlugin.success(`成功添加 ${successCount} 个视频到素材库`);
    loadVideoAssets();
}

// 删除生成的视频记录
const deleteVideo = async (video: any) => {
    const idx = generatedVideos.value.findIndex(v => v.id === video.id);
    if (idx > -1) {
        generatedVideos.value.splice(idx, 1);
        MessagePlugin.success('删除成功');
    }
}

// 从素材库中删除
const handleDeleteSource = async (asset: any) => {
    const confirm = DialogPlugin.confirm({
        header: '删除素材',
        body: '确定要从素材库中删除该视频吗？',
        onConfirm: async () => {
            try {
                const res = await deleteSource(asset.id);
                if (res.code === 0 || res.success) {
                    MessagePlugin.success('删除成功');
                    loadVideoAssets();
                } else {
                    MessagePlugin.error(res.message || res.msg || '删除失败');
                }
            } catch (e) {
                MessagePlugin.error('请求异常');
            } finally {
                confirm.destroy();
            }
        },
        onCancel: () => confirm.destroy()
    });
}

const previewImage = (url: string) => window.open(getImageUrl(url), '_blank')

onMounted(() => initData())
</script>

<style scoped lang="less">
.professional-editor {
    height: 100vh;
    display: flex;
    flex-direction: column;
    background: var(--td-bg-color-container);
    color: var(--td-text-color-primary);

    .editor-header {
        height: 56px;
        background: #fff;
        border-bottom: 1px solid var(--td-component-stroke);
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 0 16px;
        flex-shrink: 0;

        .header-title .title {
            font-weight: 700;
            color: var(--td-text-color-primary);
        }

        .status-text {
            font-size: 12px;
            color: var(--td-text-color-placeholder);
            display: flex;
            align-items: center;
            gap: 4px;
        }
    }

    .editor-main {
        flex: 1;
        display: flex;
        overflow: hidden;

        /* 左侧侧边栏 */
        .left-sidebar {
            width: 280px;
            background: #fff;
            border-right: 1px solid var(--td-component-stroke);
            display: flex;
            flex-direction: column;
            flex-shrink: 0;

            .storyboard-panel {
                flex: 1;
                min-height: 0;
                display: flex;
                flex-direction: column;
                border-bottom: 1px solid var(--td-component-stroke);

                .panel-header {
                    padding: 12px;
                    border-bottom: 1px solid var(--td-component-stroke);
                    display: flex;
                    justify-content: space-between;
                    align-items: center;

                    h3 {
                        margin: 0;
                        font-size: 14px;
                        font-weight: 600;
                        color: var(--td-text-color-primary);
                        white-space: nowrap;
                        flex-shrink: 0;
                    }
                }

                .storyboard-list {
                    flex: 1;
                    overflow-y: auto;
                    padding: 10px;
                    display: flex;
                    flex-direction: column;
                    gap: 8px;

                    .storyboard-item {
                        display: flex;
                        gap: 10px;
                        padding: 8px;
                        border-radius: 4px;
                        background: var(--td-bg-color-container);
                        border: 1px solid var(--td-component-stroke);
                        cursor: pointer;
                        transition: all 0.2s;
                        position: relative;

                        &:hover {
                            border-color: var(--td-brand-color);

                            .drag-hint {
                                opacity: 1;
                            }
                        }

                        &.active {
                            border-color: var(--td-brand-color);
                            background: var(--td-brand-color-light);
                        }

                        .shot-thumb {
                            width: 70px;
                            height: 45px;
                            background: #eee;
                            border-radius: 2px;
                            flex-shrink: 0;
                            overflow: hidden;
                            position: relative;
                            display: flex;
                            align-items: center;
                            justify-content: center;

                            .placeholder {
                                font-size: 12px;
                                color: #999;
                            }

                            .drag-hint {
                                position: absolute;
                                inset: 0;
                                background: rgba(0, 0, 0, 0.3);
                                color: #fff;
                                display: flex;
                                align-items: center;
                                justify-content: center;
                                opacity: 0;
                                transition: opacity 0.2s;
                            }
                        }

                        .shot-content {
                            flex: 1;
                            min-width: 0;
                            display: flex;
                            flex-direction: column;
                            justify-content: center;

                            .shot-title {
                                font-size: 12px;
                                font-weight: 600;
                                white-space: nowrap;
                                overflow: hidden;
                                text-overflow: ellipsis;
                            }

                            .shot-desc {
                                font-size: 10px;
                                color: var(--td-text-color-secondary);
                                margin-top: 2px;
                            }
                        }
                    }
                }
            }

            /* 素材库面板 */
            .assets-panel {
                height: 40%;
                display: flex;
                flex-direction: column;
                background: var(--td-bg-color-secondarycontainer);

                .panel-header {
                    padding: 8px 12px;
                    display: flex;
                    justify-content: space-between;
                    align-items: center;

                    &.assets-header {
                        min-height: 44px;
                        gap: 6px;
                        flex-wrap: wrap;

                        h3 {
                            flex: 1 1 100%;
                        }

                        .header-actions {
                            width: 100%;
                            justify-content: flex-end;
                        }
                    }

                    h3 {
                        margin: 0;
                        font-size: 13px;
                        color: var(--td-text-color-primary);
                        white-space: nowrap;
                        flex-shrink: 0;
                    }

                    .header-actions {
                        display: flex;
                        gap: 6px;
                        align-items: center;
                        flex-shrink: 0;

                        :deep(.t-button) {
                            white-space: nowrap;
                        }
                    }
                }

                .assets-grid {
                    flex: 1;
                    overflow-y: auto;
                    padding: 10px;
                    display: grid;
                    grid-template-columns: repeat(2, 1fr);
                    gap: 10px;
                    align-content: start;

                    .asset-item {
                        background: #fff;
                        border-radius: 4px;
                        overflow: hidden;
                        border: 1px solid transparent;
                        cursor: grab;
                        position: relative;

                        &:hover {
                            border-color: var(--td-brand-color);
                            box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);

                            .hover-overlay {
                                display: flex !important;
                            }
                        }

                        .asset-thumb {
                            height: 60px;
                            background: #000;
                            position: relative;

                            video {
                                width: 100%;
                                height: 100%;
                                object-fit: cover;
                            }

                            .duration {
                                position: absolute;
                                right: 2px;
                                bottom: 2px;
                                background: rgba(0, 0, 0, 0.6);
                                color: #fff;
                                font-size: 10px;
                                padding: 1px 3px;
                                border-radius: 2px;
                            }

                            .hover-overlay {
                                position: absolute;
                                inset: 0;
                                background: rgba(0, 0, 0, 0.5);
                                display: none;
                                align-items: center;
                                justify-content: center;
                                gap: 12px;
                                color: #fff;
                                font-size: 20px;

                                .icon-btn {
                                    cursor: pointer;
                                    transition: transform 0.2s, color 0.2s;

                                    &:hover {
                                        transform: scale(1.2);
                                    }

                                    &.danger:hover {
                                        color: var(--td-error-color);
                                    }

                                    &.primary:hover {
                                        color: var(--td-brand-color);
                                    }
                                }
                            }
                        }

                        .asset-name {
                            font-size: 10px;
                            padding: 4px;
                            white-space: nowrap;
                            overflow: hidden;
                            text-overflow: ellipsis;
                            color: var(--td-text-color-primary);
                        }
                    }
                }

                .empty-assets {
                    margin-top: 20px;
                }
            }
        }

        /* 🔴 中间工作区：被遮挡的布局 */
        .center-workspace {
            flex: 1;
            display: flex;
            flex-direction: column;
            background-color: #1e1e1e;
            overflow: hidden;

            .preview-stage {
                flex: 1;
                min-height: 0;
                /* 必须设置，防止 flex 溢出 */
                display: flex;
                justify-content: center;
                align-items: center;
                border-bottom: 1px solid #333;
                background-image: radial-gradient(#333 1px, transparent 1px);
                background-size: 20px 20px;
                position: relative;
                z-index: 1;
                /* 保持在轨道下面即可 */

                .player-container {
                    width: 100%;
                    height: 100%;
                    max-width: 90%;
                    max-height: 90%;
                    background: #000;
                    display: flex;
                    align-items: center;
                    justify-content: center;

                    .main-player {
                        width: 100%;
                        height: 100%;
                        object-fit: contain;
                        /* 确保视频自适应缩放，不溢出 */
                    }

                    .player-placeholder {
                        text-align: center;
                        color: #666;

                        p {
                            margin-top: 10px;
                            font-size: 12px;
                        }
                    }
                }
            }

            .timeline-stage {
                height: 350px;
                /* 固定轨道区的高度 */
                flex-shrink: 0;
                /* 防止被上面的 flex 区域挤压 */
                background: #252525;
                border-top: 1px solid #333;
                z-index: 10;
                position: relative;
            }
        }

        /* 右侧属性面板 */
        .edit-panel {
            width: 360px;
            background: #fff;
            border-left: 1px solid var(--td-component-stroke);
            display: flex;
            flex-direction: column;
            flex-shrink: 0;

            .edit-tabs {
                height: 100%;
                display: flex;
                flex-direction: column;

                :deep(.t-tabs__nav) {
                    flex-shrink: 0;
                }

                :deep(.t-tabs__content) {
                    flex: 1;
                    overflow: hidden;
                    display: flex;
                    flex-direction: column;
                }

                :deep(.t-tab-panel) {
                    flex: 1;
                    overflow: hidden;
                    display: flex;
                    flex-direction: column;
                }
            }

            .tab-content {
                padding: 16px;
                flex: 1;
                overflow-y: auto;
                padding-bottom: 60px;

                &::-webkit-scrollbar {
                    width: 4px;
                }

                &::-webkit-scrollbar-thumb {
                    background: #e0e0e0;
                    border-radius: 2px;
                }
            }

            .section-group {
                margin-bottom: 24px;

                .section-header {
                    display: flex;
                    justify-content: space-between;
                    align-items: center;
                    margin-bottom: 12px;
                    font-size: 13px;
                    font-weight: 600;
                    color: var(--td-text-color-primary);
                    padding-left: 8px;
                    border-left: 3px solid var(--td-brand-color);
                }
            }

            .scene-card {
                border: 1px solid var(--td-component-stroke);
                border-radius: 6px;
                overflow: hidden;

                .scene-cover {
                    height: 120px;
                    width: 100%;
                    cursor: zoom-in;
                }

                .scene-info {
                    padding: 8px 10px;
                    background: var(--td-bg-color-secondarycontainer);

                    .scene-loc {
                        font-weight: 600;
                        font-size: 13px;
                        color: var(--td-text-color-primary);
                    }

                    .scene-meta {
                        font-size: 11px;
                        color: var(--td-text-color-secondary);
                        margin-top: 2px;
                    }
                }
            }

            .empty-box {
                border: 1px dashed var(--td-component-stroke);
                border-radius: 6px;
                height: 80px;
                display: flex;
                align-items: center;
                justify-content: center;
                gap: 6px;
                cursor: pointer;
                color: var(--td-text-color-placeholder);

                &:hover {
                    border-color: var(--td-brand-color);
                    color: var(--td-brand-color);
                }
            }

            .cast-list {
                display: flex;
                flex-wrap: wrap;
                gap: 8px;

                .cast-item {
                    display: flex;
                    flex-direction: column;
                    align-items: center;
                    width: 60px;
                    position: relative;

                    .cast-name {
                        font-size: 11px;
                        margin-top: 4px;
                        color: var(--td-text-color-secondary);
                        text-align: center;
                        width: 100%;
                        white-space: nowrap;
                        overflow: hidden;
                        text-overflow: ellipsis;
                    }

                    .remove-btn {
                        position: absolute;
                        top: 0;
                        right: 0;
                        background: rgba(0, 0, 0, 0.5);
                        color: #fff;
                        border-radius: 50%;
                        width: 16px;
                        height: 16px;
                        display: flex;
                        align-items: center;
                        justify-content: center;
                        font-size: 10px;
                        cursor: pointer;
                        opacity: 0;
                        transition: opacity 0.2s;
                    }

                    &:hover .remove-btn {
                        opacity: 1;
                    }
                }
            }

            .empty-text {
                font-size: 12px;
                color: var(--td-text-color-placeholder);
                padding: 10px;
                text-align: center;
                background: var(--td-bg-color-secondarycontainer);
                border-radius: 4px;
            }

            .video-prompt-box {
                padding: 10px;
                background: var(--td-bg-color-secondarycontainer);
                border-radius: 4px;
                font-size: 12px;
                color: var(--td-text-color-secondary);
                margin-bottom: 16px;
                border: 1px solid var(--td-component-stroke);
            }

            /* 图片生成区域 */
            .grid-entry-card {
                margin-bottom: 12px;
                height: 50px;
                border: 1px dashed var(--td-brand-color);
                border-radius: 4px;
                display: flex;
                align-items: center;
                justify-content: center;
                gap: 8px;
                cursor: pointer;
                color: var(--td-brand-color);
                font-size: 13px;

                &:hover {
                    background: var(--td-brand-color-light);
                }
            }

            .image-grid-list {
                display: grid;
                grid-template-columns: repeat(2, 1fr);
                gap: 10px;

                .image-grid-item {
                    position: relative;
                    height: 100px;
                    border-radius: 4px;
                    overflow: hidden;

                    .img {
                        width: 100%;
                        height: 100%;
                    }

                    .img-overlay {
                        position: absolute;
                        inset: 0;
                        background: rgba(0, 0, 0, 0.5);
                        display: flex;
                        align-items: center;
                        justify-content: center;
                        opacity: 0;
                        transition: opacity 0.2s;
                        z-index: 5;

                        .actions-wrapper {
                            display: flex;
                            gap: 16px;
                        }

                        .icon-btn {
                            width: 32px;
                            height: 32px;
                            border-radius: 50%;
                            background: rgba(255, 255, 255, 0.2);
                            color: #fff;
                            display: flex;
                            align-items: center;
                            justify-content: center;
                            cursor: pointer;
                            transition: all 0.2s;

                            &:hover {
                                background: rgba(255, 255, 255, 0.4);
                                transform: scale(1.1);
                            }

                            &.danger:hover {
                                background: var(--td-error-color);
                            }
                        }
                    }

                    &:hover .img-overlay {
                        opacity: 1;
                    }

                    /* 动作序列特有的裁剪按钮 */
                    .crop-btn {
                        position: absolute;
                        top: 4px;
                        right: 4px;
                        background: rgba(0, 0, 0, 0.6);
                        color: #fff;
                        width: 24px;
                        height: 24px;
                        border-radius: 4px;
                        display: flex;
                        align-items: center;
                        justify-content: center;
                        cursor: pointer;
                        z-index: 10;
                        transition: all 0.2s;
                        display: none;

                        &:hover {
                            background: var(--td-brand-color);
                        }
                    }

                    &:hover .crop-btn {
                        display: flex;
                    }
                }
            }

            /* 参考图选择器样式 */
            .reference-images-section {
                margin-top: 16px;

                .frame-type-buttons {
                    margin-bottom: 12px;
                    text-align: center;
                }

                .frame-type-content {
                    background: var(--td-bg-color-secondarycontainer);
                    padding: 12px;
                    border-radius: 6px;
                }

                .previous-frame-section {
                    margin-bottom: 12px;
                }

                .reference-grid {
                    display: grid;
                    grid-template-columns: repeat(3, 1fr);
                    gap: 8px;

                    .reference-item-mini {
                        position: relative;
                        height: 70px;
                        border-radius: 4px;
                        overflow: hidden;
                        cursor: pointer;
                        border: 2px solid transparent;
                        transition: all 0.2s;

                        &.selected {
                            border-color: var(--td-brand-color);
                            box-shadow: 0 0 0 1px var(--td-brand-color);
                        }

                        &:hover {
                            border-color: var(--td-brand-color);
                        }

                        .img {
                            width: 100%;
                            height: 100%;
                        }

                        .check-mark {
                            position: absolute;
                            top: 0;
                            right: 0;
                            background: var(--td-brand-color);
                            color: #fff;
                            border-bottom-left-radius: 4px;
                            padding: 2px 4px;
                            display: flex;
                            align-items: center;
                            justify-content: center;
                            font-size: 12px;
                        }

                        .preview-icon {
                            position: absolute;
                            bottom: 4px;
                            right: 4px;
                            background: rgba(0, 0, 0, 0.5);
                            color: #fff;
                            border-radius: 4px;
                            padding: 4px;
                            display: flex;
                            align-items: center;
                            justify-content: center;
                            opacity: 0;
                            transition: opacity 0.2s;
                        }

                        &:hover .preview-icon {
                            opacity: 1;
                        }
                    }
                }

                /* 顶部的占位槽位样式 */
                .first-last-slots {
                    display: flex;
                    justify-content: center;
                    align-items: center;
                    gap: 24px;
                    margin-top: 16px;
                }

                .ref-container {
                    display: flex;
                    gap: 10px;

                    &.center {
                        justify-content: center;
                    }

                    &.row {
                        justify-content: space-between;
                        align-items: center;
                    }

                    .ref-container-column {
                        display: flex;
                        flex-direction: column;
                        gap: 10px;
                    }
                }

                .slot-wrapper {
                    display: flex;
                    flex-direction: column;
                    align-items: center;
                    gap: 4px;
                }

                .ref-label {
                    font-size: 13px;
                    font-weight: 500;
                    color: var(--td-text-color-primary);
                }

                .ref-image-wrapper {
                    position: relative;
                    display: inline-block;

                    .ref-delete-btn {
                        position: absolute;
                        top: -6px;
                        right: -6px;
                        color: var(--td-error-color);
                        background: #fff;
                        border-radius: 50%;
                        cursor: pointer;
                        z-index: 10;
                        display: flex;
                        transition: transform 0.2s;
                        box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);

                        &:hover {
                            transform: scale(1.1);
                        }
                    }
                }

                .ref-image-slot {
                    width: 140px;
                    height: 80px;
                    border: 2px dashed var(--td-component-stroke);
                    border-radius: 6px;
                    overflow: hidden;
                    cursor: pointer;
                    display: flex;
                    align-items: center;
                    justify-content: center;
                    background: var(--td-bg-color-container);
                    transition: border-color 0.2s;

                    &.slot-small {
                        width: 80px;
                        height: 50px;
                    }

                    &.selected {
                        border-color: var(--td-brand-color);
                        border-style: solid;
                    }

                    .placeholder {
                        color: #ccc;
                        transition: color 0.2s;
                        display: flex;
                        flex-direction: column;
                        align-items: center;
                    }

                    &:hover .placeholder {
                        color: var(--td-brand-color);
                    }

                    &:hover {
                        border-color: var(--td-brand-color);
                    }
                }
            }

            /* 视频列表 */
            .video-card-list {
                display: flex;
                flex-direction: column;
                gap: 12px;

                .video-card {
                    background: #000;
                    border-radius: 4px;
                    overflow: hidden;
                    border: 1px solid var(--td-component-stroke);
                    position: relative;

                    .video-thumbnail {
                        position: relative;
                        width: 100%;
                        height: 150px;
                        cursor: pointer;
                        background: #111;

                        video {
                            width: 100%;
                            height: 100%;
                            object-fit: cover;
                            display: block;
                        }

                        .play-overlay {
                            position: absolute;
                            inset: 0;
                            display: flex;
                            align-items: center;
                            justify-content: center;
                            background: rgba(0, 0, 0, 0.2);
                            transition: background 0.3s;

                            &:hover {
                                background: rgba(0, 0, 0, 0.4);
                            }
                        }
                    }

                    .video-placeholder {
                        width: 100%;
                        height: 150px;
                        display: flex;
                        flex-direction: column;
                        align-items: center;
                        justify-content: center;
                        background: #2a2a2a;
                    }

                    .video-actions {
                        padding: 8px;
                        display: flex;
                        justify-content: space-between;
                        align-items: center;
                        background: #fff;
                        border-top: 1px solid var(--td-component-stroke);

                        .action-btns {
                            display: flex;
                            gap: 4px;
                        }
                    }
                }
            }

            /* 合成记录 */
            .merge-list {
                display: flex;
                flex-direction: column;
                gap: 8px;

                .merge-item {
                    display: flex;
                    justify-content: space-between;
                    align-items: center;
                    padding: 10px;
                    background: var(--td-bg-color-container);
                    border-radius: 4px;

                    .merge-info {
                        .title {
                            font-size: 13px;
                            font-weight: 500;
                        }

                        .time {
                            font-size: 11px;
                            color: #999;
                        }
                    }
                }
            }
        }
    }
}

.char-selector-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 16px;
    padding: 10px;

    .char-item {
        display: flex;
        flex-direction: column;
        align-items: center;
        cursor: pointer;
        border: 2px solid transparent;
        padding: 10px;
        border-radius: 8px;
        position: relative;

        &:hover {
            background: var(--td-bg-color-secondarycontainer);
        }

        &.selected {
            border-color: var(--td-brand-color);
            background: var(--td-brand-color-light);
        }

        span {
            margin-top: 8px;
            font-size: 12px;
            font-weight: 500;
        }

        .check {
            position: absolute;
            top: 8px;
            right: 8px;
            color: var(--td-brand-color);
        }
    }
}
</style>
