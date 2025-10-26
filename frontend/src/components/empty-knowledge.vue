<script setup lang="ts">
import { ref } from 'vue';
import DocsiteImportDialog from './docsite-import-dialog.vue';
import CreateKnowledgeDialog from './create-knowledge-dialog.vue';

const props = defineProps<{
  kbId?: string;
}>();

const emit = defineEmits<{
  (e: 'refresh'): void;
}>();

const showDocsiteDialog = ref(false);
const showCreateDialog = ref(false);

const handleImportClick = () => {
  showDocsiteDialog.value = true;
};

const handleCreateClick = () => {
  showCreateDialog.value = true;
};

const handleImportSuccess = () => {
  emit('refresh');
};

const handleCreateSuccess = () => {
  emit('refresh');
};
</script>
<template>
    <div class="empty">
        <img class="empty-img" src="@/assets/img/upload.svg" alt="">
        <span class="empty-txt">知识为空，拖放上传</span>
        <span class="empty-type-txt">pdf、doc 格式文件，不超过10M</span>
        <span class="empty-type-txt">text、markdown格式文件，不超过200K</span>
        <div class="import-actions">
          <span class="divider-text">或者</span>
          <div class="action-buttons">
            <t-button theme="primary" @click="handleCreateClick">
              <t-icon name="add" />
              创建知识
            </t-button>
            <t-button theme="default" variant="outline" @click="handleImportClick">
              <t-icon name="link" />
              从文档站导入
            </t-button>
          </div>
        </div>
        <CreateKnowledgeDialog 
          v-model:visible="showCreateDialog" 
          :kb-id="kbId || ''"
          @success="handleCreateSuccess"
        />
        <DocsiteImportDialog 
          v-model:visible="showDocsiteDialog" 
          :kb-id="kbId || ''"
          @success="handleImportSuccess"
        />
    </div>
</template>
<style scoped lang="less">
.empty {
    flex: 1;
    display: flex;
    flex-flow: column;
    justify-content: center;
    align-items: center;
}

.empty-txt {
    color: #00000099;
    font-family: "PingFang SC";
    font-size: 16px;
    font-weight: 600;
    line-height: 26px;
    margin: 12px 0 16px 0;
}

.empty-type-txt {
    color: #00000066;
    text-align: center;
    font-family: "PingFang SC";
    font-size: 12px;
    font-weight: 400;
    width: 217px;
}

.empty-img {
    width: 162px;
    height: 162px;
}

.import-actions {
    margin-top: 24px;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;

    .divider-text {
        color: #00000066;
        font-size: 14px;
        margin: 8px 0;
    }

    .action-buttons {
        display: flex;
        gap: 12px;
        align-items: center;
        
        :deep(.t-button) {
            display: inline-flex;
            align-items: center;
            justify-content: center;
            
            .t-icon {
                display: inline-flex;
                align-items: center;
            }
        }
    }
}
</style>
