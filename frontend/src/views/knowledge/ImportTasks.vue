<script setup lang="ts">
import { ref, onMounted, computed } from 'vue';
import { useRoute } from 'vue-router';
import { MessagePlugin } from 'tdesign-vue-next';
import { listImportTasks, getImportTask, cancelImportTask } from '@/api/knowledge-base';

const route = useRoute();
const kbId = computed(() => route.params.kbId as string);

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

const tasks = ref<ImportTask[]>([]);
const loading = ref(false);
const pagination = ref({
  current: 1,
  pageSize: 10,
  total: 0
});

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
    width: 300,
    ellipsis: true
  },
  {
    colKey: 'status',
    title: '状态',
    width: 100,
    cell: (h: any, { row }: any) => {
      const status = statusMap[row.status] || { label: row.status, theme: 'default' };
      return h('t-tag', { theme: status.theme, variant: 'light' }, status.label);
    }
  },
  {
    colKey: 'progress',
    title: '进度',
    width: 250,
    cell: (h: any, { row }: any) => {
      if (row.total_urls === 0) return h('span', '-');
      const percentage = ((row.processed_urls / row.total_urls) * 100).toFixed(0);
      return h('div', { style: 'display: flex; align-items: center; gap: 8px;' }, [
        h('t-progress', {
          percentage: Number(percentage),
          size: 'small',
          style: 'flex: 1'
        }),
        h('span', { style: 'white-space: nowrap;' }, `${row.processed_urls}/${row.total_urls}`)
      ]);
    }
  },
  {
    colKey: 'stats',
    title: '统计',
    width: 150,
    cell: (h: any, { row }: any) => {
      return h('div', { style: 'font-size: 12px;' }, [
        h('div', `成功: ${row.success_count}`),
        h('div', `失败: ${row.failed_count}`),
        h('div', `重复: ${row.duplicate_count}`)
      ]);
    }
  },
  {
    colKey: 'created_at',
    title: '创建时间',
    width: 180,
    cell: (h: any, { row }: any) => {
      return h('span', formatDateTime(row.created_at));
    }
  },
  {
    colKey: 'operation',
    title: '操作',
    width: 120,
    cell: (h: any, { row }: any) => {
      const canCancel = row.status === 'pending' || row.status === 'processing';
      return h('t-space', [
        h('t-link', {
          theme: 'primary',
          onClick: () => handleRefresh(row.id)
        }, '刷新'),
        canCancel && h('t-link', {
          theme: 'danger',
          onClick: () => handleCancel(row.id)
        }, '取消')
      ]);
    }
  }
];

const loadTasks = async () => {
  loading.value = true;
  try {
    const result = await listImportTasks({
      page: pagination.value.current,
      page_size: pagination.value.pageSize,
      knowledge_base_id: kbId.value
    });
    
    if (result) {
      const data = result as any;
      tasks.value = data.data || [];
      pagination.value.total = data.total || 0;
    }
  } catch (error: any) {
    console.error('Load import tasks error:', error);
    MessagePlugin.error('加载导入任务列表失败');
  } finally {
    loading.value = false;
  }
};

const handleRefresh = async (taskId: string) => {
  try {
    const result = await getImportTask(taskId);
    if (result) {
      const data = result as any;
      if (data.success && data.data) {
        const index = tasks.value.findIndex(t => t.id === taskId);
        if (index !== -1) {
          tasks.value[index] = data.data;
        }
        MessagePlugin.success('任务状态已更新');
      }
    }
  } catch (error: any) {
    console.error('Refresh task error:', error);
    MessagePlugin.error('刷新任务状态失败');
  }
};

const handleCancel = async (taskId: string) => {
  try {
    const result = await cancelImportTask(taskId);
    if (result) {
      const data = result as any;
      if (data.success) {
        MessagePlugin.success('任务已取消');
        await loadTasks();
      } else {
        MessagePlugin.error(data.message || '取消任务失败');
      }
    }
  } catch (error: any) {
    console.error('Cancel task error:', error);
    MessagePlugin.error('取消任务失败');
  }
};

const handlePageChange = (pageInfo: any) => {
  pagination.value.current = pageInfo.current;
  pagination.value.pageSize = pageInfo.pageSize;
  loadTasks();
};

const formatDateTime = (dateStr: string) => {
  if (!dateStr) return '-';
  const date = new Date(dateStr);
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  });
};

// 自动刷新处理中的任务
let refreshInterval: any = null;
const startAutoRefresh = () => {
  refreshInterval = setInterval(() => {
    const hasProcessingTasks = tasks.value.some(
      t => t.status === 'pending' || t.status === 'processing'
    );
    if (hasProcessingTasks) {
      loadTasks();
    }
  }, 5000); // 每5秒刷新一次
};

onMounted(() => {
  loadTasks();
  startAutoRefresh();
});

// 清理定时器
import { onUnmounted } from 'vue';
onUnmounted(() => {
  if (refreshInterval) {
    clearInterval(refreshInterval);
  }
});
</script>

<template>
  <div class="import-tasks-page">
    <div class="page-header">
      <h2>批量导入任务</h2>
      <t-button theme="primary" @click="loadTasks" :loading="loading">
        <template #icon><t-icon name="refresh" /></template>
        刷新列表
      </t-button>
    </div>

    <t-table
      :data="tasks"
      :columns="columns"
      :loading="loading"
      row-key="id"
      :pagination="pagination"
      @page-change="handlePageChange"
      stripe
      hover
    >
      <template #empty>
        <div class="empty-state">
          <t-icon name="inbox" size="48px" style="color: #dcdcdc;" />
          <p>暂无导入任务</p>
        </div>
      </template>
    </t-table>
  </div>
</template>

<style scoped lang="less">
.import-tasks-page {
  padding: 24px;

  .page-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 24px;

    h2 {
      margin: 0;
      font-size: 20px;
      font-weight: 600;
      color: #000000e6;
    }
  }

  .empty-state {
    padding: 48px 0;
    text-align: center;

    p {
      margin-top: 12px;
      color: #00000066;
      font-size: 14px;
    }
  }
}
</style>
