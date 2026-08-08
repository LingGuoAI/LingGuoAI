<template>
    <t-dialog v-model:visible="visible" header="AI 服务配置" width="1000px" placement="center" :footer="false"
        destroy-on-close>
        <div class="ai-config-container">
            <div class="header-actions">
                <t-tabs v-model="activeTab" @change="handleTabChange">
                    <t-tab-panel value="text" label="文本模型" />
                    <t-tab-panel value="image" label="图片模型" />
                    <t-tab-panel value="video" label="视频模型" />
                </t-tabs>
                <t-button theme="primary" @click="showCreateDialog">
                    <template #icon><plus-icon /></template>
                    添加配置
                </t-button>
            </div>

            <t-table row-key="id" :data="configs" :columns="columns" :loading="loading" hover stripe
                style="margin-top: 16px;">
                <template #model="{ row }">
                    <t-space size="small" wrap>
                        <t-tag v-for="m in row.model" :key="m" theme="primary" variant="light">{{ m }}</t-tag>
                    </t-space>
                </template>
                <template #is_active="{ row }">
                    <t-switch v-model="row.is_active" :custom-value="[1, 0]" @change="handleToggleActive(row)" />
                </template>
                <template #op="{ row }">
                    <t-space size="small">
                        <t-button variant="text" theme="primary" @click="handleTest(row)">测试</t-button>
                        <t-button variant="text" theme="primary" @click="handleEdit(row)">编辑</t-button>
                        <t-button variant="text" theme="danger" @click="handleDelete(row)">删除</t-button>
                    </t-space>
                </template>
            </t-table>
        </div>
    </t-dialog>

    <t-dialog v-model:visible="dialogVisible" :header="isEdit ? '编辑配置' : '添加配置'" width="650px" placement="center"
        attach="body" destroy-on-close :confirm-btn="submitting ? '保存中...' : (isEdit ? '保存' : '创建')"
        @confirm="handleSubmit">
        <t-form ref="formRef" :data="form" :rules="rules" label-width="120px" label-align="right">
            <t-form-item label="配置名称" name="name">
                <t-input v-model="form.name" placeholder="请输入配置名称，如：OpenAI-Text" />
            </t-form-item>

            <t-form-item label="厂商提供商" name="provider">
                <t-select v-model="form.provider" placeholder="请选择服务提供商" @change="handleProviderChange">
                    <t-option v-for="provider in availableProviders" :key="provider.id" :label="provider.name"
                        :value="provider.id" :disabled="provider.disabled" />
                </t-select>
                <t-comment help="目前可用的厂商类型" />
            </t-form-item>

            <t-form-item label="优先级" name="priority">
                <t-input-number v-model="form.priority" :min="0" :max="100" :step="1" style="width: 100%" />
                <t-comment help="数字越大优先级越高" />
            </t-form-item>

            <t-form-item label="支持的模型" name="model">
                <t-select v-model="form.model" placeholder="请选择或输入模型名后回车" multiple creatable filterable
                    :min-collapsed-num="3">
                    <t-option v-for="model in availableModels" :key="model" :label="model" :value="model" />
                </t-select>
                <t-comment help="支持多选，可输入自定义模型名称后按回车添加（⚠️火山引擎请填写控制台接入点ID）" />
            </t-form-item>

            <t-form-item label="接口地址" name="base_url">
                <t-input v-model="form.base_url" placeholder="https://api.example.com/v1" />
                <template #help>
                    包含域名和协议的完整基础路径。<br />
                    实际请求路径示例: {{ fullEndpointExample }}
                </template>
            </t-form-item>

            <t-form-item label="API Key" name="api_key">
                <t-input v-model="form.api_key" type="password" clearable placeholder="sk-..." />
            </t-form-item>

            <t-form-item v-if="isEdit" label="状态" name="is_active">
                <t-switch v-model="form.is_active" :custom-value="[1, 0]" />
            </t-form-item>

            <t-form-item style="margin-top: 24px;">
                <t-button variant="outline" theme="default" :loading="testing" @click="testConnection">
                    测试连通性
                </t-button>
            </t-form-item>
        </t-form>
    </t-dialog>

    <t-dialog v-model:visible="testResultVisible" header="测试结果" width="600px" placement="center" :footer="false"
        attach="body" @close="stopPolling">
        <div v-if="testPolling" class="polling-container">
            <t-loading size="medium" text="AI 正在处理中，请耐心稍候..." />
        </div>
        <div v-else class="result-content">
            <div v-if="testResultType === 'text'" class="text-result">
                {{ testResultData }}
            </div>
            <div v-else-if="testResultType === 'image'" class="image-result">
                <t-image :src="testResultData" fit="contain"
                    style="width: 100%; max-height: 400px; border-radius: 8px;" />
            </div>
            <div v-else-if="testResultType === 'video'" class="video-result">
                <video :src="testResultData" controls autoplay playsinline loop
                    style="width: 100%; max-height: 400px; border-radius: 8px; outline: none; background: #000;">
                    您的浏览器不支持视频播放。
                </video>
            </div>
            <div v-else class="text-result">
                {{ testResultData }}
            </div>
        </div>
    </t-dialog>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch } from 'vue';
import { MessagePlugin, DialogPlugin, FormRule } from 'tdesign-vue-next';
import { PlusIcon } from 'tdesign-icons-vue-next';
import { request } from "@/utils/request";

import {
    getAiConfigList,
    createAiConfig,
    updateAiConfig,
    deleteAiConfig,
    testTextConfig,
    testImageConfig,
    testVideoConfig
} from '@/api/ai_config';

import type { AIServiceType } from '@/types/ai';
import { getImageUrl } from '@/utils/format';

const props = defineProps({
    visible: Boolean
});

const emit = defineEmits(['update:visible']);

const visible = computed({
    get: () => props.visible,
    set: (val) => emit('update:visible', val)
});

// --- 列表与核心状态 ---
const activeTab = ref<AIServiceType>('text');
const loading = ref(false);
const configs = ref<any[]>([]);

const columns = [
    { colKey: 'name', title: '配置名称', width: 150 },
    { colKey: 'provider', title: '厂商', width: 120 },
    { colKey: 'model', title: '模型列表', width: 280 },
    { colKey: 'priority', title: '优先级', width: 100, align: 'center' },
    { colKey: 'is_active', title: '状态', width: 100 },
    { colKey: 'op', title: '操作', width: 200, fixed: 'right' }
];

// --- 表单与弹框状态 ---
const dialogVisible = ref(false);
const isEdit = ref(false);
const editingId = ref<number>();
const formRef = ref<any>(null);
const submitting = ref(false);
const testing = ref(false);

const form = reactive({
    service_type: 'text',
    provider: '',
    name: '',
    base_url: '',
    api_key: '',
    model: [],
    priority: 0,
    is_active: 1,
});

// --- 常量配置 ---
interface ProviderConfig {
    id: string;
    name: string;
    models: string[];
    disabled?: boolean;
}

const providerConfigs: Record<string, ProviderConfig[]> = {
    text: [
        { id: 'openai', name: 'OpenAI', models: ['gpt-4o', 'gpt-4-turbo'] },
        { id: 'getgoapi', name: 'NewRouter API', models: ['gemini-3-flash-preview', 'gpt-5.4'] },
        { id: 'gemini', name: 'Google Gemini', models: ['gemini-1.5-pro', 'gemini-3-flash-preview'] },
        { id: 'doubao', name: '火山引擎', models: ['doubao-pro-32k', 'doubao-lite-32k'] },
    ],
    image: [
        { id: 'volcengine', name: '火山引擎', models: ['doubao-seedream-4-5-251128', 'doubao-seedream-4-0-250828'] },
        { id: 'getgoapi', name: 'NewRouter API', models: ['doubao-seedream-4-5-251128', 'dall-e-3'] },
        { id: 'openai', name: 'OpenAI', models: ['dall-e-3'] },
    ],
    video: [
        { id: 'volces', name: '火山引擎', models: ['doubao-seedance-1-5-pro-251215'] },
        { id: 'openai', name: 'OpenAI Sora', models: ['sora-2'] },
        { id: 'minimax', name: '海螺 MiniMax', models: ['MiniMax-Hailuo-02'] },
        { id: 'kling', name: '可灵 Kling', models: ['kling'] },
        { id: 'runway', name: 'Runway', models: ['runway'] },
        { id: 'pika', name: 'Pika', models: ['pika'] },
        { id: 'google', name: 'Google Veo', models: ['veo-3.1-fast-generate-001'] },
        { id: 'getgoapi', name: 'NewRouter API', models: ['doubao-seedance-1-5-pro-251215', 'sora-2', 'MiniMax-Hailuo-02'] },
    ],
};

const availableProviders = computed(() => providerConfigs[form.service_type] || []);
const availableModels = computed(() => {
    if (!form.provider) return [];
    const providerDef = availableProviders.value.find(p => p.id === form.provider);
    return providerDef ? providerDef.models : [];
});

const fullEndpointExample = computed(() => {
    const baseUrl = form.base_url || 'https://api.example.com';
    const provider = form.provider;
    const serviceType = form.service_type;
    let endpoint = '';
    if (serviceType === 'text') {
        endpoint = (provider === 'gemini') ? '/v1beta/models/{model}:generateContent' : '/chat/completions';
    } else if (serviceType === 'image') {
        endpoint = (provider === 'gemini') ? '/v1beta/models/{model}:generateContent' : '/images/generations';
    } else if (serviceType === 'video') {
        endpoint = '/videos';
    }
    return baseUrl + endpoint;
});

const rules: Record<string, FormRule[]> = {
    name: [{ required: true, message: '请输入配置名称' }],
    provider: [{ required: true, message: '请选择厂商' }],
    base_url: [{ required: true, message: '请输入 Base URL' }],
    api_key: [{ required: true, message: '请输入 API Key' }],
    model: [{ required: true, message: '请选择模型' }]
};

// --- 核心业务方法 ---
const loadConfigs = async () => {
    loading.value = true;
    try {
        const res: any = await getAiConfigList({ service_type: activeTab.value, per_page: 100 });
        configs.value = res.data?.list || [];
    } finally { loading.value = false; }
};

watch(visible, (val) => { if (val) loadConfigs(); });

const handleTabChange = (value: AIServiceType) => {
    activeTab.value = value;
    loadConfigs();
};

const generateConfigName = (provider: string, serviceType: AIServiceType) => {
    const providerNames: Record<string, string> = { getgoapi: 'GetGo', openai: 'OpenAI', gemini: 'Gemini', doubao: 'Volc', volces: 'Volc', volcengine: 'Volc' };
    const serviceNames: Record<AIServiceType, string> = { text: '文本', image: '图片', video: '视频' };
    const randomNum = Math.floor(Math.random() * 10000).toString().padStart(4, '0');
    return `${providerNames[provider] || provider}-${serviceNames[serviceType] || serviceType}-${randomNum}`;
};

const resetForm = () => {
    Object.assign(form, { service_type: activeTab.value, provider: '', name: '', base_url: '', api_key: '', model: [], priority: 0, is_active: 1 });
    formRef.value?.clearValidate();
};

const showCreateDialog = () => {
    isEdit.value = false;
    editingId.value = undefined;
    resetForm();
    form.provider = 'getgoapi';
    form.base_url = 'https://www.zdom.cn/v1';
    form.name = generateConfigName('getgoapi', activeTab.value);
    dialogVisible.value = true;
};

const handleEdit = (config: any) => {
    isEdit.value = true;
    editingId.value = config.id;
    Object.assign(form, {
        ...config,
        model: Array.isArray(config.model) ? config.model : [config.model],
    });
    dialogVisible.value = true;
};

// 🔴 已在此处完善所有服务类型及提供商的自动化切 URL 逻辑
const handleProviderChange = () => {
    form.model = [];

    const defaultUrls: Record<AIServiceType, Record<string, string>> = {
        text: {
            openai: 'https://api.openai.com/v1',
            getgoapi: 'https://www.zdom.cn/v1',
            gemini: 'https://generativelanguage.googleapis.com/v1beta',
            doubao: 'https://ark.cn-beijing.volces.com/api/v3',
        },
        image: {
            volcengine: 'https://ark.cn-beijing.volces.com/api/v3',
            getgoapi: 'https://www.zdom.cn/v1',
            openai: 'https://api.openai.com/v1',
        },
        video: {
            volces: 'https://ark.cn-beijing.volces.com/api/v3',
            openai: 'https://api.openai.com/v1',
            minimax: 'https://api.minimax.chat/v1',
            kling: 'https://api.klingai.com/v1',
            runway: 'https://api.runwayml.com/v1',
            pika: 'https://api.pika.art/v1',
            google: 'https://generativelanguage.googleapis.com/v1beta',
            getgoapi: 'https://www.zdom.cn/v1',
        }
    };

    const currentServiceType = form.service_type as AIServiceType;
    if (defaultUrls[currentServiceType] && defaultUrls[currentServiceType][form.provider]) {
        form.base_url = defaultUrls[currentServiceType][form.provider];
    } else {
        form.base_url = '';
    }

    if (!isEdit.value) {
        form.name = generateConfigName(form.provider, form.service_type as AIServiceType);
    }
};

const handleSubmit = async () => {
    const validateResult = await formRef.value.validate();
    if (validateResult !== true) return;
    submitting.value = true;
    try {
        isEdit.value ? await updateAiConfig(editingId.value!, form) : await createAiConfig(form);
        MessagePlugin.success('操作成功');
        dialogVisible.value = false;
        loadConfigs();
    } finally { submitting.value = false; }
};

const handleToggleActive = async (config: any) => {
    try {
        await updateAiConfig(config.id, { is_active: config.is_active });
        MessagePlugin.success('状态已更新');
    } catch (e) {
        config.is_active = config.is_active === 1 ? 0 : 1;
    }
};

const handleDelete = (config: any) => {
    DialogPlugin.confirm({
        header: '警告',
        body: `确定要删除配置 [${config.name}] 吗？`,
        theme: 'danger',
        onConfirm: async () => {
            await deleteAiConfig(config.id);
            loadConfigs();
        }
    });
};

// ==========================================
// 测试与轮询逻辑 
// ==========================================
const testResultVisible = ref(false);
const testPolling = ref(false);
const testResultType = ref('');
const testResultData = ref('');
let pollTimer: any = null;

const stopPolling = () => { if (pollTimer) { clearTimeout(pollTimer); pollTimer = null; } };

const startPolling = (taskId: number | string, type: string) => {
    testResultVisible.value = true;
    testPolling.value = true;
    testResultType.value = type;
    testResultData.value = '';

    const poll = async () => {
        try {
            const res: any = await request.get({ url: `/tasks/${taskId}` });
            const payload = res.data?.data || res.data;
            const task = payload?.task || payload;
            const statusName = task?.statusName || task?.status_name;

            if ((task.status === 2 || statusName === 'succeeded') && task.process === 100) {
                testPolling.value = false;
                let resultObj: any = {};
                try { resultObj = JSON.parse(task.result || '{}'); } catch (e) { resultObj = { raw: task.result }; }

                if (type === 'text') {
                    testResultData.value = resultObj.reply || task.result;
                } else {
                    const rawUrl = resultObj.video_url || resultObj.url || resultObj.image_url;
                    testResultData.value = getImageUrl(rawUrl);
                }
                MessagePlugin.success('测试成功！');
                stopPolling();
            } else if (task.status === 4 || task.status === -1 || task.status === 3 || statusName === 'failed' || statusName === 'cancelled') {
                testPolling.value = false;
                testResultData.value = task.errorMsg || task.error_msg || '任务失败';
                MessagePlugin.error('测试执行失败');
                stopPolling();
            } else {
                pollTimer = setTimeout(poll, 3000);
            }
        } catch (e) {
            testPolling.value = false;
            stopPolling();
        }
    };
    poll();
};

const executeTest = async (payload: any, serviceType: string) => {
    testing.value = true;
    stopPolling();
    try {
        let res: any;
        if (serviceType === 'text') res = await testTextConfig(payload);
        else if (serviceType === 'image') res = await testImageConfig(payload);
        else if (serviceType === 'video') res = await testVideoConfig(payload);

        if (res.data?.reply || res.data?.image_url) {
            testResultVisible.value = true;
            testPolling.value = false;
            testResultType.value = serviceType;
            testResultData.value = serviceType === 'text' ? res.data.reply : getImageUrl(res.data.image_url);
            MessagePlugin.success('测试成功！');
        } else if (res.data?.task_id) {
            MessagePlugin.info('任务已提交，正在生成...');
            startPolling(res.data.task_id, serviceType);
        }
    } catch (e: any) {
        MessagePlugin.error(e.message || '请求失败');
    } finally { testing.value = false; }
};

const testConnection = () => {
    executeTest({ ...form }, form.service_type as AIServiceType);
};

const handleTest = (config: any) => {
    executeTest({ ...config }, config.service_type as AIServiceType);
};
</script>

<style scoped>
.header-actions {
    display: flex;
    justify-content: space-between;
    align-items: center;
    border-bottom: 1px solid var(--td-component-stroke);
    padding-bottom: 12px;
}

.polling-container {
    display: flex;
    justify-content: center;
    align-items: center;
    min-height: 150px;
}

.result-content {
    min-height: 100px;
    padding: 16px;
    background: var(--td-bg-color-container-active);
    border-radius: 8px;
    display: flex;
    justify-content: center;
    align-items: center;
}

.text-result {
    font-size: 14px;
    line-height: 1.6;
    white-space: pre-wrap;
    word-break: break-all;
    width: 100%;
}
</style>
