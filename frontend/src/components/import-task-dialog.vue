<template>
  <t-dialog
    v-model:visible="visible"
    header="批量导入任务"
    :width="900"
    :footer="false"
    @close="handleClose"
  >
    <div class="import-task-dialog">
      <t-table
        :data="tasks"
        :columns="columns"
        :loading="loading"
        :pagination="pagination"
        @page-change="handlePageChange"
        row-key="id"
        stripe
        hover
      >
        <template #empty>
          <div class="empty-state">
            <p>暂无导入任务</p>
          </div>
        </template>
      </t-table>
    </div>
  </t-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue';
import { MessagePlugin } from 'tdesign-vue-next';
import { listImportTasks, cancelImportTask } from '@/api/knowledge-base';

interface Props {
  modelValue: boolean;
  knowledgeBaseId: string;
}

interface ImportTask {
  id: string;
  base_url: string;
  status: string;
  total_urls: number;
  processed_urls: number;
  success_count: number;
  failed_count: number;
  duplicate_count: number;
  current_url: string;
  created_at: string;
  updated_at: string;
  completed_at?: string;
  error_message?: string;
}

const props = defineProps<Props>();
const emit = defineEmits(['update:modelValue']);

const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
});

const tasks = ref<ImportTask[]>([]);
const loading = ref(false);
const pagination = ref({
  current: 1,
  pageSize: 10,
  total: 0
});

let refreshTimer: ReturnType<typeof setInterval> | null = null;

const statusMap: Record<string, { label: string; theme: string }> = {
  pending: { label: '等待中', theme: 'default' },
  processing: { label: '进行中', theme: 'primary' },
  completed: { label: '已完成', theme: 'success' },
  failed: { label: '失败', theme: 'danger' },
  cancelled: { label: '已取消', theme: 'warning' }
};

const columns = [
  {
    colKey: 'base_url',
    title: '文档站地址',
    width: 250,
    ellipsis: true
  },
  {
    colKey: 'status',
    title: '状态',
    width: 80,
    cell: (h: any, { row }: any) => {
      const status = statusMap[row.status] || { label: row.status, theme: 'default' };
      return h('t-tag', { theme: status.theme, variant: 'light', size: 'small' }, status.label);
    }
  },
  {
    colKey: 'progress',
    title: '进度',
    width: 200,
    cell: (h: any, { row }: any) => {
      if (row.total_urls === 0) return h('span', '-');
      const percentage = ((row.processed_urls / row.total_urls) * 100).toFixed(0);
      return h('div', { style: 'display: flex; align-items: center; gap: 8px;' }, [
        h('t-progress', {
          percentage: Number(percentage),
          size: 'small',
          style: 'flex: 1; min-width: 80px;'
        }),
        h('span', { style: 'white-space: nowrap; font-size: 12px;' }, `${row.processed_urls}/${row.total_urls}`)
      ]);
    }
  },
  {
    colKey: 'stats',
    title: '统计',
    width: 120,
    cell: (h: any, { row }: any) => {
      return h('div', { style: 'font-size: 12px; line-height: 1.5;' }, [
        h('div', `✓ ${row.success_count}`),
        row.failed_count > 0 ? h('div', { style: 'color: var(--td-error-color);' }, `✗ ${row.failed_count}`) : null,
        row.duplicate_count > 0 ? h('div', { style: 'color: var(--td-warning-color);' }, `⊚ ${row.duplicate_count}`) : null
      ]);
    }
  },
  {
    colKey: 'created_at',
    title: '创建时间',
    width: 150,
    cell: (h: any, { row }: any) => {
      return h('span', { style: 'font-size: 12px;' }, formatDateTime(row.created_at));
    }
  },
  {
    colKey: 'operation',
    title: '操作',
    width: 80,
    cell: (h: any, { row }: any) => {
      const canCancel = row.status === 'pending' || row.status === 'processing';
      if (!canCancel) return null;
      
      return h('t-button', {
        theme: 'danger',
        variant: 'text',
        size: 'small',
        onClick: () => handleCancel(row.id)
      }, '取消');
    }
  }
];

const formatDateTime = (dateStr: string) => {
  if (!dateStr) return '-';
  const date = new Date(dateStr);
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  
  if (diff < 60000) return '刚刚';
  if (diff < 3600000) return `${Math.floor(diff / 60000)}分钟前`;
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}小时前`;
  
  return date.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  });
};

const fetchTasks = async () => {
  if (!props.knowledgeBaseId) return;
  
  loading.value = true;
  try {
    const result = await listImportTasks({
      knowledge_base_id: props.knowledgeBaseId,
      page: pagination.value.current,
      page_size: pagination.value.pageSize
    });
    
    tasks.value = result.data || [];
    pagination.value.total = result.total || 0;
  } catch (error: any) {
    console.error('Failed to load import tasks:', error);
    MessagePlugin.error(error.message || '加载任务列表失败');
  } finally {
    loading.value = false;
  }
};

const handlePageChange = (pageInfo: any) => {
  pagination.value.current = pageInfo.current;
  pagination.value.pageSize = pageInfo.pageSize;
  fetchTasks();
};

const handleCancel = async (taskId: string) => {
  try {
    await cancelImportTask(taskId);
    MessagePlugin.success('任务已取消');
    fetchTasks();
  } catch (error: any) {
    console.error('Failed to cancel task:', error);
    MessagePlugin.error(error.message || '取消任务失败');
  }
};

const handleClose = () => {
  if (refreshTimer) {
    clearInterval(refreshTimer);
    refreshTimer = null;
  }
};

const startAutoRefresh = () => {
  if (refreshTimer) return;
  
  refreshTimer = setInterval(() => {
    const hasActiveTask = tasks.value.some(
      task => task.status === 'pending' || task.status === 'processing'
    );
    
    if (hasActiveTask) {
      fetchTasks();
    }
  }, 3000);
};

watch(() => props.modelValue, (newValue) => {
  if (newValue) {
    fetchTasks();
    startAutoRefresh();
  } else {
    handleClose();
  }
});

watch(() => tasks.value, () => {
  const hasActiveTask = tasks.value.some(
    task => task.status === 'pending' || task.status === 'processing'
  );
  
  if (!hasActiveTask && refreshTimer) {
    clearInterval(refreshTimer);
    refreshTimer = null;
  } else if (hasActiveTask && !refreshTimer && visible.value) {
    startAutoRefresh();
  }
}, { deep: true });
</script>

<style scoped>
.import-task-dialog {
  padding: 4px 0;
}

.empty-state {
  text-align: center;
  padding: 40px 0;
  color: var(--td-text-color-placeholder);
}
</style>
