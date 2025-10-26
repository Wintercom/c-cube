<script setup lang="ts">
import { ref, computed } from 'vue';
import { MessagePlugin } from 'tdesign-vue-next';
import { createKnowledgeFromURL, uploadKnowledgeFile } from '@/api/knowledge-base';

const props = defineProps<{
  visible: boolean;
  kbId: string;
}>();

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void;
  (e: 'success'): void;
}>();

const createType = ref<'url' | 'file'>('url');
const url = ref('');
const isCreating = ref(false);
const fileInputRef = ref<HTMLInputElement>();
const selectedFiles = ref<File[]>([]);

const dialogVisible = computed({
  get: () => props.visible,
  set: (val) => emit('update:visible', val)
});

const isValidUrl = computed(() => {
  if (!url.value) return false;
  try {
    new URL(url.value);
    return true;
  } catch {
    return false;
  }
});

const canSubmit = computed(() => {
  if (createType.value === 'url') {
    return isValidUrl.value && !isCreating.value;
  } else {
    return selectedFiles.value.length > 0 && !isCreating.value;
  }
});

const handleCreate = async () => {
  if (!props.kbId) {
    MessagePlugin.warning('知识库ID不存在');
    return;
  }

  isCreating.value = true;

  try {
    if (createType.value === 'url') {
      await handleUrlCreate();
    } else {
      await handleFileUpload();
    }
  } catch (error: any) {
    console.error('Create knowledge error:', error);
    MessagePlugin.error(error?.message || '创建知识时发生错误');
  } finally {
    isCreating.value = false;
  }
};

const handleUrlCreate = async () => {
  if (!isValidUrl.value) {
    MessagePlugin.warning('请输入有效的URL地址');
    return;
  }

  try {
    const result = await createKnowledgeFromURL(props.kbId, {
      url: url.value,
      enable_multimodel: false
    });

    if (result) {
      const data = result as any;
      if (data.success || data.data) {
        MessagePlugin.success('URL导入成功，正在后台解析...');
        handleClose();
        emit('success');
      } else {
        MessagePlugin.error(data.message || '创建知识失败');
      }
    }
  } catch (error: any) {
    throw error;
  }
};

const handleFileUpload = async () => {
  if (selectedFiles.value.length === 0) {
    MessagePlugin.warning('请选择要上传的文件');
    return;
  }

  try {
    const formData = new FormData();
    selectedFiles.value.forEach(file => {
      formData.append('file', file);
    });

    const result = await uploadKnowledgeFile(props.kbId, formData);
    
    if (result) {
      const data = result as any;
      if (data.success || data.data) {
        MessagePlugin.success('文件上传成功，正在后台解析...');
        
        window.dispatchEvent(new CustomEvent('knowledgeFileUploaded', {
          detail: { kbId: props.kbId }
        }));
        
        handleClose();
        emit('success');
      } else {
        MessagePlugin.error(data.message || '文件上传失败');
      }
    }
  } catch (error: any) {
    throw error;
  }
};

const handleFileSelect = () => {
  fileInputRef.value?.click();
};

const onFileChange = (event: Event) => {
  const target = event.target as HTMLInputElement;
  if (target.files && target.files.length > 0) {
    selectedFiles.value = Array.from(target.files);
  }
};

const removeFile = (index: number) => {
  selectedFiles.value.splice(index, 1);
};

const handleClose = () => {
  url.value = '';
  selectedFiles.value = [];
  createType.value = 'url';
  emit('update:visible', false);
};
</script>

<template>
  <t-dialog
    v-model:visible="dialogVisible"
    header="创建知识"
    :confirm-btn="{
      content: isCreating ? '创建中...' : '确认创建',
      theme: 'primary',
      disabled: !canSubmit,
      loading: isCreating
    }"
    :cancel-btn="{ content: '取消', disabled: isCreating }"
    @confirm="handleCreate"
    @close="handleClose"
    width="520px"
  >
    <div class="create-knowledge-form">
      <t-radio-group v-model="createType" class="type-selector">
        <t-radio-button value="url">从URL创建</t-radio-button>
        <t-radio-button value="file">上传文件</t-radio-button>
      </t-radio-group>

      <div v-if="createType === 'url'" class="url-section">
        <div class="form-item">
          <label class="form-label">URL地址<span class="required">*</span></label>
          <t-input
            v-model="url"
            placeholder="https://example.com/document"
            :disabled="isCreating"
            clearable
          />
          <span class="form-hint">请输入要导入的网页URL地址</span>
        </div>
      </div>

      <div v-else class="file-section">
        <div class="form-item">
          <label class="form-label">选择文件<span class="required">*</span></label>
          <div class="file-upload-area">
            <t-button
              theme="default"
              variant="outline"
              @click="handleFileSelect"
              :disabled="isCreating"
              class="file-select-button"
            >
              <t-icon name="upload" />
              选择文件
            </t-button>
            <input
              ref="fileInputRef"
              type="file"
              multiple
              accept=".pdf,.doc,.docx,.txt,.md"
              style="display: none"
              @change="onFileChange"
            />
          </div>
          <span class="form-hint">支持 pdf、doc、docx、txt、md 格式，单个文件不超过30M</span>
          
          <div v-if="selectedFiles.length > 0" class="file-list">
            <div v-for="(file, index) in selectedFiles" :key="index" class="file-item">
              <t-icon name="file" class="file-icon" />
              <span class="file-name">{{ file.name }}</span>
              <span class="file-size">({{ (file.size / 1024 / 1024).toFixed(2) }}MB)</span>
              <t-icon
                name="close"
                class="remove-icon"
                @click="removeFile(index)"
                v-if="!isCreating"
              />
            </div>
          </div>
        </div>
      </div>

      <div v-if="isCreating" class="creating-status">
        <t-loading size="small" />
        <span class="status-text">正在创建知识，请稍候...</span>
      </div>
    </div>
  </t-dialog>
</template>

<style scoped lang="less">
.create-knowledge-form {
  padding: 8px 0;

  .type-selector {
    margin-bottom: 24px;
  }

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

  .file-upload-area {
    margin-bottom: 8px;
    
    .file-select-button {

      :deep(.t-button__text) {
        display: inline-flex;
        align-items: center;
      }

      :deep(.t-icon) {
        display: inline-flex;
        align-items: center;
        margin-right: 4px;
      }
    }
  }

  .file-list {
    margin-top: 12px;
    border: 1px solid #e7e7e7;
    border-radius: 4px;
    padding: 8px;
    max-height: 200px;
    overflow-y: auto;

    .file-item {
      display: flex;
      align-items: center;
      padding: 8px;
      border-radius: 4px;
      margin-bottom: 4px;
      background: #f9f9f9;

      &:last-child {
        margin-bottom: 0;
      }

      .file-icon {
        color: #0052d9;
        margin-right: 8px;
        flex-shrink: 0;
      }

      .file-name {
        flex: 1;
        color: #000000e6;
        font-size: 14px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }

      .file-size {
        color: #00000066;
        font-size: 12px;
        margin-left: 8px;
        flex-shrink: 0;
      }

      .remove-icon {
        color: #00000066;
        margin-left: 8px;
        cursor: pointer;
        flex-shrink: 0;

        &:hover {
          color: #e34d59;
        }
      }
    }
  }

  .creating-status {
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
