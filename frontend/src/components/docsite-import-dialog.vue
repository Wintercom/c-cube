<script setup lang="ts">
import { ref, computed } from 'vue';
import { MessagePlugin } from 'tdesign-vue-next';
import { createImportTask } from '@/api/knowledge-base';
import { useRouter } from 'vue-router';

const props = defineProps<{
  visible: boolean;
  kbId: string;
}>();

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void;
  (e: 'success'): void;
}>();

const router = useRouter();
const baseUrl = ref('');
const maxPages = ref(100);
const isImporting = ref(false);
const showAdvanced = ref(false);

const dialogVisible = computed({
  get: () => props.visible,
  set: (val) => emit('update:visible', val)
});

const isValidUrl = computed(() => {
  if (!baseUrl.value) return false;
  try {
    new URL(baseUrl.value);
    return true;
  } catch {
    return false;
  }
});

const handleImport = async () => {
  if (!isValidUrl.value) {
    MessagePlugin.warning('请输入有效的文档站地址');
    return;
  }

  if (!props.kbId) {
    MessagePlugin.warning('知识库ID不存在');
    return;
  }

  isImporting.value = true;

  try {
    const result = await createImportTask(props.kbId, {
      base_url: baseUrl.value,
      max_pages: maxPages.value,
      enable_multimodel: false
    });

    if (result) {
      const data = result as any;
      if (data.success && data.data) {
        const taskId = data.data.id;
        MessagePlugin.success('导入任务已创建,正在后台处理...');
        handleClose();
        
        // 跳转到任务管理页面
        router.push({
          name: 'ImportTasks',
          params: { kbId: props.kbId }
        });
      } else {
        MessagePlugin.error(data.message || '创建导入任务失败');
      }
    }
  } catch (error: any) {
    console.error('Create import task error:', error);
    MessagePlugin.error(error?.message || '创建导入任务时发生错误');
  } finally {
    isImporting.value = false;
  }
};

const handleClose = () => {
  baseUrl.value = '';
  maxPages.value = 100;
  showAdvanced.value = false;
  emit('update:visible', false);
};
</script>

<template>
  <t-dialog
    v-model:visible="dialogVisible"
    header="批量导入文档站"
    :confirm-btn="{
      content: isImporting ? '导入中...' : '开始导入',
      theme: 'primary',
      disabled: isImporting || !isValidUrl,
      loading: isImporting
    }"
    :cancel-btn="{ content: '取消', disabled: isImporting }"
    @confirm="handleImport"
    @close="handleClose"
    width="520px"
  >
    <div class="docsite-import-form">
      <div class="form-item">
        <label class="form-label">文档站地址<span class="required">*</span></label>
        <t-input
          v-model="baseUrl"
          placeholder="https://docs.example.com"
          :disabled="isImporting"
          clearable
        />
        <span class="form-hint">请输入完整的文档站URL地址</span>
      </div>

      <div class="advanced-toggle" @click="showAdvanced = !showAdvanced">
        <t-icon :name="showAdvanced ? 'chevron-down' : 'chevron-right'" />
        <span>高级配置</span>
      </div>

      <div v-show="showAdvanced" class="advanced-section">
        <div class="form-item">
          <label class="form-label">最大页面数</label>
          <t-input-number
            v-model="maxPages"
            :min="1"
            :max="500"
            :disabled="isImporting"
            theme="normal"
          />
          <span class="form-hint">限制爬取的最大页面数量 (1-500)</span>
        </div>
      </div>

      <div v-if="isImporting" class="importing-status">
        <t-loading size="small" />
        <span class="status-text">正在导入文档站,请稍候...</span>
      </div>
    </div>
  </t-dialog>
</template>

<style scoped lang="less">
.docsite-import-form {
  padding: 8px 0;

  .form-item {
    margin-bottom: 20px;

    .form-label {
      display: block;
      margin-bottom: 8px;
      color: #000000e6;
      font-size: 14px;
      font-weight: 500;

      .required {
        color: #e34d59;
        margin-left: 2px;
      }
    }

    .form-hint {
      display: block;
      margin-top: 4px;
      color: #00000066;
      font-size: 12px;
    }
  }

  .advanced-toggle {
    display: flex;
    align-items: center;
    margin-bottom: 16px;
    color: #0052d9;
    font-size: 14px;
    cursor: pointer;
    user-select: none;

    &:hover {
      opacity: 0.8;
    }

    span {
      margin-left: 4px;
    }
  }

  .advanced-section {
    padding-left: 20px;
    border-left: 2px solid #f3f3f3;
  }

  .importing-status {
    display: flex;
    align-items: center;
    padding: 12px;
    background: #f3f9ff;
    border-radius: 4px;
    margin-top: 16px;

    .status-text {
      margin-left: 12px;
      color: #0052d9;
      font-size: 14px;
    }
  }
}
</style>
