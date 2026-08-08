<template>
  <div class="workflow-page">
    <div class="page-header">
      <t-button variant="text" shape="circle" @click="router.back()"><template #icon><t-icon name="arrow-left" /></template></t-button>
      <div><h2>{{ workflow?.title || 'AI 剧本创作' }}</h2><div class="muted">剧本创作师 v1.1 · 四步创作流程</div></div>
      <t-tag :theme="workflow?.status === 'completed' ? 'success' : 'primary'">{{ statusText(workflow?.status) }}</t-tag>
    </div>
    <t-card v-if="!workflowId" title="创建剧本工作流" :bordered="false">
      <t-form :data="form" label-align="top" @submit="createWorkflow">
        <div class="form-grid">
          <t-form-item label="工作流名称"><t-input v-model="form.title" placeholder="例如：第11个嫌疑人" /></t-form-item>
          <t-form-item label="创作起点"><t-select v-model="form.startingPoint" :options="startingOptions" /></t-form-item>
          <t-form-item label="剧本类型"><t-select v-model="form.genre" :options="genreOptions" /></t-form-item>
          <t-form-item label="总集数"><t-input-number v-model="form.totalEpisodes" :min="5" :max="30" /></t-form-item>
          <t-form-item label="目标受众"><t-radio-group v-model="form.targetAudience"><t-radio value="男频">男频</t-radio><t-radio value="女频">女频</t-radio><t-radio value="通用">通用</t-radio></t-radio-group></t-form-item>
          <t-form-item label="单集时长（秒）"><t-input-number v-model="form.episodeDuration" :min="60" :max="180" /></t-form-item>
        </div>
        <t-form-item label="核心故事"><t-textarea v-model="form.coreStory" :autosize="{ minRows: 5 }" placeholder="一句话或完整描述均可，零创意时可以留空" /></t-form-item>
        <t-form-item label="补充要求"><t-textarea v-model="form.requirements" /></t-form-item>
        <t-button theme="primary" type="submit" :loading="busy">创建并开始</t-button>
      </t-form>
    </t-card>
    <template v-else>
      <t-card :bordered="false"><t-steps :current="Math.max(0,(workflow?.currentStep || 1)-1)" readonly><t-step-item title="创意定型" content="类型、集数与核心创意"/><t-step-item title="全剧设计" content="大纲、角色与世界观"/><t-step-item title="剧本生成" content="每批最多5集"/><t-step-item title="AI交付" content="场景、角色与统计"/></t-steps></t-card>
      <t-card class="stage-card" :bordered="false" :title="stageTitle">
        <template #actions><t-button variant="text" @click="loadDetail">刷新</t-button></template>
        <div v-if="currentStep === 1"><p>系统将根据创作表单整理并补全创意，生成可确认的 v1.0 状态。</p><JsonResult :value="stepOutput('creative_lock')"/><StageActions stage="creative_lock" :ready="stepReady('creative_lock')" @run="runStage" @confirm="confirmStep"/></div>
        <div v-else-if="currentStep === 2"><p>生成三幕结构、逐集大纲、伏笔系统、角色、世界观和核心冲突。</p><JsonResult :value="stepOutput('complete_design')"/><StageActions stage="complete_design" :ready="stepReady('complete_design')" @run="runStage" @confirm="confirmStep"/></div>
        <div v-else-if="currentStep === 3">
          <div class="batch-bar"><t-radio-group v-model="mode"><t-radio-button value="preview">快速预览</t-radio-button><t-radio-button value="single">单集生成</t-radio-button><t-radio-button value="batch">批量生成</t-radio-button></t-radio-group><t-input-number v-model="startEpisode" :min="1" :max="workflow.totalEpisodes"/><span>至</span><t-input-number v-model="endEpisode" :min="1" :max="workflow.totalEpisodes"/><t-button theme="primary" :loading="busy" @click="generateEpisodes">开始生成</t-button></div>
          <div class="episode-list"><t-collapse v-model="openEpisodes"><t-collapse-panel v-for="ep in episodeRows" :key="ep.episodeNo" :value="ep.episodeNo" :header="`第 ${ep.episodeNo} 集 · ${ep.title || '待生成'}`"><template #headerRightContent><t-tag :theme="episodeTheme(ep.status)">{{ episodeStatus(ep.status) }}</t-tag></template><t-alert v-if="qualityWarnings(ep).length" theme="warning" :message="qualityWarnings(ep).join('；')"/><t-textarea v-if="ep.contentText" v-model="ep.contentText" :autosize="{minRows:12}"/><div class="episode-actions"><t-button variant="outline" :loading="busy" @click="generateOne(ep.episodeNo, '')">重新生成</t-button><t-button variant="outline" :disabled="!ep.contentText" @click="saveEpisode(ep)">保存修改</t-button><t-button theme="success" :disabled="ep.status !== 'waiting_confirmation'" @click="confirmEpisode(ep.episodeNo)">确认并发布</t-button></div></t-collapse-panel></t-collapse></div>
        </div>
        <div v-else><p>所有剧本已确认。生成去重场景、角色清单、镜头统计和视觉风格建议。</p><JsonResult :value="stepOutput('ai_delivery')"/><StageActions stage="ai_delivery" :ready="stepReady('ai_delivery')" @run="runStage" @confirm="confirmStep"/></div>
      </t-card>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, reactive, ref, resolveComponent } from 'vue';
import { MessagePlugin } from 'tdesign-vue-next';
import { useRoute, useRouter } from 'vue-router';
import { confirmWorkflowEpisode, confirmWorkflowStep, createScriptWorkflow, generateWorkflowEpisodes, getScriptWorkflow, runWorkflowStage, updateWorkflowEpisode } from '@/api/script-workflows';
import { findTasks } from '@/api/tasks';

const router=useRouter(); const route=useRoute(); const workflowId=ref<any>(route.params.workflowId || ''); const workflow=ref<any>(); const steps=ref<any[]>([]); const episodes=ref<any[]>([]); const busy=ref(false); const mode=ref('preview'); const startEpisode=ref(1); const endEpisode=ref(1); const openEpisodes=ref<any[]>([]);
const form=reactive({title:'',startingPoint:'已有完整创意',genre:'悬疑推理',totalEpisodes:10,targetAudience:'通用',episodeDuration:90,coreStory:'',requirements:''});
const startingOptions=['完全零创意','有模糊想法','已有完整创意','从其他工具衔接'].map(v=>({label:v,value:v})); const genreOptions=['都市爱情','霸总复仇','古装宫斗','玄幻修仙','悬疑推理','搞笑沙雕','现代职场','末世求生'].map(v=>({label:v,value:v}));
const currentStep=computed(()=>workflow.value?.currentStep||1); const stageTitle=computed(()=>['','步骤1 · 创意定型','步骤2 · 全剧设计','步骤3 · 剧本生成','步骤4 · AI交付'][currentStep.value]);
const episodeRows=computed(()=>Array.from({length:workflow.value?.totalEpisodes||0},(_,i)=>episodes.value.find(e=>e.episodeNo===i+1)||{episodeNo:i+1,status:'pending'}));
const latestStep=(key:string)=>steps.value.filter(s=>s.stepKey===key).sort((a,b)=>b.version-a.version)[0]; const stepOutput=(key:string)=>{const step=latestStep(key);return step?.outputJson||step?.outputText||''}; const stepReady=(key:string)=>!!stepOutput(key);
const JsonResult=defineComponent({props:{value:String},setup(p){return()=>p.value?h('pre',{class:'json-result'},formatJSON(p.value)):h('div',{class:'empty-result'},'尚未生成')}}); const StageActions=defineComponent({props:{stage:String,ready:Boolean},emits:['run','confirm'],setup(p,{emit}){const Button=resolveComponent('t-button');const Space=resolveComponent('t-space');return()=>h(Space,{class:'stage-actions'},()=>[h(Button,{theme:'primary',onClick:()=>emit('run',p.stage)},()=>p.ready?'重新生成':'开始生成'),h(Button,{theme:'success',disabled:!p.ready,onClick:()=>emit('confirm',p.stage)},()=>p.stage==='ai_delivery'?'确认交付并完成':'确认并进入下一步')])}});
function unwrap(res:any){return res.data?.data||res.data} function formatJSON(v?:string){try{return JSON.stringify(JSON.parse(v||''),null,2)}catch{return v||''}}
async function createWorkflow(){busy.value=true;try{const res:any=await createScriptWorkflow({projectId:Number(route.params.projectId),title:form.title||'AI 剧本创作',config:{...form}});const data=unwrap(res);workflowId.value=data.id;await router.replace(`/admin/projects/${route.params.projectId}/script-workflow/${data.id}`);await loadDetail()}finally{busy.value=false}}
async function loadDetail(){if(!workflowId.value)return;const res:any=await getScriptWorkflow(workflowId.value);const data=unwrap(res);workflow.value=data.workflow;steps.value=data.steps||[];episodes.value=data.episodes||[]}
async function waitTasks(ids:number[]){
  await Promise.all(ids.map(id=>new Promise<void>((resolve,reject)=>{
    let attempts=0;
    const timer=setInterval(async()=>{
      try {
        const r:any=await findTasks(id); const payload=unwrap(r); const t=payload?.task||payload; attempts++;
        if(t?.status===2||t?.statusName==='succeeded'){clearInterval(timer);resolve()}
        else if(t?.status===3||t?.statusName==='failed'){clearInterval(timer);reject(new Error(t.errorMsg||t.error||'任务失败'))}
        else if(t?.status===4||t?.statusName==='cancelled'){clearInterval(timer);reject(new Error('任务已取消'))}
        else if(attempts>=150){clearInterval(timer);reject(new Error('任务等待超时，请查看 Worker 日志'))}
      } catch(e){ if(attempts>=5){clearInterval(timer);reject(e)} }
    },1800)
  })));
  await loadDetail()
}
async function runStage(stage:string){busy.value=true;try{const r:any=await runWorkflowStage(workflowId.value,{stepKey:stage});await waitTasks([unwrap(r).taskId]);MessagePlugin.success('生成完成，请确认结果')}catch(e:any){MessagePlugin.error(e.message||'生成失败')}finally{busy.value=false}}
async function confirmStep(stage:string){try{busy.value=true;const res:any=await confirmWorkflowStep(workflowId.value,stage);if(res.code!==undefined&&res.code!==0&&res.success!==true)throw new Error(res.message||'确认失败');await loadDetail();MessagePlugin.success(stage==='ai_delivery'?'交付已确认，工作流完成':'已确认，已进入下一步')}catch(e:any){MessagePlugin.error(e.message||'确认失败')}finally{busy.value=false}}
async function generateEpisodes(){let s=startEpisode.value,e=endEpisode.value;if(mode.value==='preview'){s=1;e=1}else if(mode.value==='single'){e=s}if(e<s||e-s+1>5)return MessagePlugin.warning('每批最多5集');busy.value=true;try{const r:any=await generateWorkflowEpisodes(workflowId.value,{startEpisode:s,endEpisode:e});await waitTasks(unwrap(r).taskIds);MessagePlugin.success('本批剧本生成完成')}catch(e:any){MessagePlugin.error(e.message||'生成失败')}finally{busy.value=false}}
async function generateOne(ep:number,changeRequest:string){busy.value=true;try{const r:any=await runWorkflowStage(workflowId.value,{stepKey:'episode_generation',episodeNo:ep,changeRequest});await waitTasks([unwrap(r).taskId])}finally{busy.value=false}}
async function saveEpisode(ep:any){await updateWorkflowEpisode(workflowId.value,ep.episodeNo,ep.contentText);await loadDetail();MessagePlugin.success('已保存版本')}
async function confirmEpisode(ep:number){await confirmWorkflowEpisode(workflowId.value,ep);await loadDetail();MessagePlugin.success('已发布到剧本列表')}
const qualityWarnings=(ep:any)=>{try{return JSON.parse(ep.qualityReportJson||'{}').warnings||[]}catch{return[]}}; const statusText=(s:string)=>({draft:'草稿',running:'生成中',waiting_confirmation:'待确认',completed:'已完成',failed:'失败'} as any)[s]||s; const episodeStatus=(s:string)=>({pending:'待生成',queued:'排队中',generating:'生成中',waiting_confirmation:'待确认',confirmed:'已发布',failed:'失败'} as any)[s]||s; const episodeTheme=(s:string)=>s==='confirmed'?'success':s==='failed'?'danger':s==='waiting_confirmation'?'warning':'default';
onMounted(loadDetail);
</script>

<style scoped lang="less">
.workflow-page{padding:24px;max-width:1400px;margin:auto}.page-header{display:flex;align-items:center;gap:16px;margin-bottom:20px}.page-header h2{margin:0}.muted{color:var(--td-text-color-secondary)}.form-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:0 24px}.stage-card{margin-top:16px}.json-result{max-height:520px;overflow:auto;padding:16px;background:var(--td-bg-color-secondarycontainer);border-radius:8px;white-space:pre-wrap}.empty-result{padding:48px;text-align:center;color:var(--td-text-color-placeholder)}.stage-actions,.episode-actions{margin-top:16px}.batch-bar{display:flex;align-items:center;gap:12px;flex-wrap:wrap;margin-bottom:16px}.episode-list{margin-top:12px}.episode-actions{display:flex;gap:12px}@media(max-width:768px){.form-grid{grid-template-columns:1fr}.workflow-page{padding:12px}}
</style>
