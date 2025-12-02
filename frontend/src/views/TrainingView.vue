<template>
  <div class="training-view">
    <!-- 顶部栏 - 系统资源监控 -->
    <div class="top-bar">
      <!-- CPU 监控 -->
      <div class="resource-card">
        <div class="resource-icon">
          <Cpu :size="24" />
        </div>
        <div class="resource-info">
          <div class="progress-bar">
            <div class="progress-fill success" :style="{ width: systemStats.cpuUsage + '%' }"></div>
          </div>
          <div class="resource-text">
            {{ t('training.systemResources.cpu.cores', { count: systemStats.cpuCores }) }}
          </div>
        </div>
      </div>

      <!-- 内存监控 -->
      <div class="resource-card">
        <div class="resource-icon">
          <MemoryStick :size="24" />
        </div>
        <div class="resource-info">
          <div class="progress-bar">
            <div class="progress-fill" :class="systemStats.memUsage > 85 ? 'error' : 'success'" :style="{ width: systemStats.memUsage + '%' }"></div>
          </div>
          <div class="resource-text">{{ systemStats.memUsed }}GB / {{ systemStats.memTotal }}GB</div>
        </div>
      </div>

      <!-- GPU 监控 -->
      <div class="resource-card">
        <div class="resource-icon">
          <Monitor :size="24" />
        </div>
        <div class="resource-info">
          <template v-if="systemStats.gpuAvailable">
            <div class="progress-bar">
              <div class="progress-fill warning" :style="{ width: systemStats.gpuUsage + '%' }"></div>
            </div>
            <div class="resource-text">
              {{ t('training.systemResources.gpu.vram') }}: {{ systemStats.vramUsed }}GB / {{ systemStats.vramTotal }}GB
            </div>
          </template>
          <div v-else class="resource-text not-available">
            {{ t('training.systemResources.gpu.notAvailable') }}
          </div>
        </div>
      </div>

      <!-- 训练插件状态 -->
      <div class="resource-card">
        <div class="resource-icon">
          <Puzzle :size="24" />
        </div>
        <div class="resource-info">
          <template v-if="trainingPluginReady">
            <div class="resource-status">
              <span class="status-text deployed">{{ t('training.systemResources.plugin.ready') }}</span>
              <span class="help-icon" :title="t('training.systemResources.plugin.readyHint')">
                <HelpCircle :size="14" />
              </span>
            </div>
          </template>
          <template v-else>
            <div class="resource-status">
              <span class="status-text not-deployed">{{ t('training.systemResources.plugin.notReady') }}</span>
              <span class="help-icon" :title="t('training.systemResources.plugin.notReadyHint')">
                <HelpCircle :size="14" />
              </span>
            </div>
            <button class="btn btn-small btn-primary" @click="navigateToPythonEnv">
              {{ t('training.systemResources.plugin.setupButton') }}
            </button>
          </template>
        </div>
      </div>
    </div>

    <!-- 主内容区域 -->
    <div class="main-content">
      <!-- 左侧面板 -->
      <div class="left-panel">
        <!-- 上半部分 - 训练任务 -->
        <div class="tasks-section" :style="{ height: `calc(100% - ${bottomHeight}px)` }">
          <div class="section-header">
            <span class="section-title">{{ t('training.tasks.title') }}</span>
            <div class="header-actions">
              <button class="btn btn-small" @click="showTrainsetDialog = true">
                <FolderPlus :size="14" />
                {{ t('training.trainset.createButton') }}
              </button>
              <button class="btn btn-small btn-primary" @click="showNewTaskDialog = true">
                <Plus :size="14" />
                {{ t('training.tasks.newTask') }}
              </button>
            </div>
          </div>
          <div class="tasks-body">
            <div v-if="trainingTasks.length > 0" class="task-list">
              <div v-for="task in trainingTasks" :key="task.id" class="task-card">
                <div class="task-header">
                  <div class="task-name-row">
                    <span class="task-name">{{ task.name }}</span>
                    <span :class="['task-badge', `badge-${task.status}`]">
                      <Loader2 v-if="task.status === 'running'" :size="12" class="animate-spin" />
                      <Clock v-else-if="task.status === 'pending'" :size="12" />
                      <X v-else-if="task.status === 'error'" :size="12" />
                      <Check v-else-if="task.status === 'completed'" :size="12" />
                      {{ t(`training.tasks.status.${task.status}`) }}
                    </span>
                  </div>
                  <div class="task-menu-wrapper">
                    <button class="btn-icon" @click="toggleTaskMenu(task.id)">
                      <MoreHorizontal :size="16" />
                    </button>
                    <div v-if="openMenuTaskId === task.id" class="task-menu">
                      <button v-if="task.status === 'running'" @click="handleTaskAction('stop', task)">{{ t('training.tasks.menu.stop') }}</button>
                      <template v-else>
                        <button @click="handleTaskAction('delete', task)">{{ t('training.tasks.menu.delete') }}</button>
                        <button @click="handleTaskAction('retrain', task)">{{ t('training.tasks.menu.retrain') }}</button>
                      </template>
                    </div>
                  </div>
                </div>
                <div class="task-progress-row">
                  <span class="epoch-text">{{ t('training.tasks.epoch') }}: {{ task.currentEpoch }}/{{ task.totalEpochs }}</span>
                  <span class="progress-percent">{{ task.batchProgress.toFixed(2) }}%</span>
                </div>
                <div class="progress-bar small">
                  <div :class="['progress-fill', task.status === 'error' ? 'error' : 'success']" :style="{ width: task.batchProgress + '%' }"></div>
                </div>
                <div class="task-metrics">
                  <div class="metrics-group">
                    <div class="metric-item"><span class="metric-label">{{ t('training.tasks.metrics.boxLoss') }}</span><span class="metric-value">{{ task.metrics.boxLoss?.toFixed(4) || '-' }}</span></div>
                    <div class="metric-item"><span class="metric-label">{{ t('training.tasks.metrics.clsLoss') }}</span><span class="metric-value">{{ task.metrics.clsLoss?.toFixed(4) || '-' }}</span></div>
                    <div class="metric-item"><span class="metric-label">{{ t('training.tasks.metrics.trainLoss') }}</span><span class="metric-value">{{ task.metrics.trainLoss?.toFixed(4) || '-' }}</span></div>
                  </div>
                  <div class="metrics-group">
                    <div class="metric-item"><span class="metric-label">{{ t('training.tasks.metrics.mAP50') }}</span><span class="metric-value">{{ task.metrics.mAP50?.toFixed(4) || '-' }}</span></div>
                    <div class="metric-item"><span class="metric-label">{{ t('training.tasks.metrics.mAP5095') }}</span><span class="metric-value">{{ task.metrics.mAP5095?.toFixed(4) || '-' }}</span></div>
                    <div class="metric-item"><span class="metric-label">{{ t('training.tasks.metrics.valLoss') }}</span><span class="metric-value">{{ task.metrics.valLoss?.toFixed(4) || '-' }}</span></div>
                  </div>
                </div>
              </div>
            </div>
            <div v-else class="empty-state">
              <FileText :size="48" class="empty-icon" />
              <p class="empty-text">{{ t('training.tasks.empty') }}</p>
              <p class="empty-hint">{{ t('training.tasks.emptyHint') }}</p>
            </div>
          </div>
        </div>
        <div class="resize-handle" @mousedown="startResize"></div>
        <!-- 下半部分 - 终端窗口 -->
        <div class="logs-section" :style="{ height: `${bottomHeight}px` }">
          <div class="section-header">
            <span class="section-title">{{ t('training.logs.title') }}</span>
            <div class="log-actions">
              <button class="btn-icon" @click="copyTerminalContent" :title="t('training.logs.copy')"><Copy :size="14" /></button>
              <button class="btn-icon" @click="clearTerminal" :title="t('training.logs.clear')"><Trash2 :size="14" /></button>
            </div>
          </div>
          <div ref="terminalContainerRef" class="terminal-container"></div>
        </div>
      </div>
      <!-- 右侧面板 - 训练历史 -->
      <div class="right-panel">
        <div class="section-header"><span class="section-title">{{ t('training.history.title') }}</span></div>
        <div class="history-body">
          <div v-if="trainingHistory.length > 0" class="history-list">
            <div v-for="record in trainingHistory" :key="record.id" class="history-item">
              <div class="history-header">
                <div class="history-name">{{ record.name }}</div>
                <div class="history-menu-wrapper">
                  <button class="btn-icon" @click="toggleHistoryMenu(record.id)">
                    <MoreHorizontal :size="14" />
                  </button>
                  <div v-if="openHistoryMenuId === record.id" class="history-menu">
                    <button @click="handleHistoryAction('open', record)">
                      <FolderOpen :size="14" />{{ t('training.history.openFolder') }}
                    </button>
                    <button @click="handleHistoryAction('delete', record)">
                      <Trash2 :size="14" />{{ t('training.history.delete') }}
                    </button>
                  </div>
                </div>
              </div>
              <div class="history-meta"><span class="history-time">{{ record.completedAt }}</span></div>
            </div>
          </div>
          <div v-else class="empty-state">
            <History :size="32" class="empty-icon" />
            <p class="empty-text">{{ t('training.history.empty') }}</p>
            <p class="empty-hint">{{ t('training.history.emptyHint') }}</p>
          </div>
        </div>
      </div>
    </div>

    <!-- 管理依赖对话框 -->
    <div v-if="showEnvDialog" class="modal-overlay" @click.self="showEnvDialog = false">
      <div class="modal env-modal">
        <h2 class="modal-title">{{ t('training.envDialog.title') }}</h2>
        <div class="env-dialog-content">
          <div class="env-left">
            <div class="env-row">
              <label class="env-label">{{ t('training.envDialog.pythonVersion') }}</label>
              <div class="env-value">
                <span v-if="pythonEnv.pythonVersion">{{ pythonEnv.pythonVersion }}</span>
                <span v-else class="text-warning">{{ t('training.envDialog.pythonNotInstalled') }}</span>
                <span v-if="isPythonVersionWarning" class="warning-tag">{{ t('training.envDialog.pythonVersionWarning') }}</span>
              </div>
            </div>
            <div class="env-row">
              <label class="env-label">{{ t('training.envDialog.venvPath') }}</label>
              <div class="env-value">
                <span v-if="pythonEnv.deployed" class="venv-path">{{ pythonEnv.venvPath }}</span>
                <span v-else class="text-muted">{{ t('training.envDialog.venvNotDeployed') }}</span>
              </div>
            </div>
            <template v-if="!pythonEnv.deployed">
              <div class="env-row">
                <button class="btn btn-primary btn-env-action" :disabled="!pythonEnv.pythonVersion || isDeploying" @click="deployEnvironment">
                  <Loader2 v-if="isDeploying" :size="16" class="animate-spin" />
                  {{ isDeploying ? t('training.envDialog.deploying') : t('training.systemResources.python.deployButton') }}
                </button>
              </div>
            </template>
            <template v-else>
              <div class="env-row">
                <label class="env-label">{{ t('training.envDialog.pypiMirror') }}</label>
                <div class="mirror-row">
                  <label class="checkbox-inline">
                    <input type="checkbox" v-model="usePypiMirror" />
                    <span>{{ t('training.envDialog.enableMirror') }}</span>
                  </label>
                  <input type="text" v-model="pypiMirrorUrl" :disabled="!usePypiMirror" class="input mirror-input" :placeholder="t('training.envDialog.mirrorPlaceholder')" />
                </div>
              </div>
              <div class="env-row">
                <label class="env-label">{{ t('training.envDialog.installPackage') }}</label>
                <div class="install-input-row">
                  <input type="text" v-model="packageToInstall" :placeholder="t('training.envDialog.packagePlaceholder')" :disabled="isInstalling" class="input" @keyup.enter="installPackage" />
                  <button class="btn btn-primary btn-install" :disabled="!packageToInstall.trim() || isInstalling" @click="installPackage">
                    <Loader2 v-if="isInstalling" :size="14" class="animate-spin" />{{ t('training.envDialog.install') }}
                  </button>
                </div>
              </div>
              <div class="env-row env-row-actions">
                <button class="btn btn-danger btn-env-action" :disabled="isUndeploying" @click="showUndeployConfirm = true">
                  <Trash2 :size="14" />{{ t('training.envDialog.undeployButton') }}
                </button>
              </div>
            </template>
          </div>
          <div class="env-right">
            <label class="env-label">{{ t('training.envDialog.installedPackages') }}</label>
            <div class="packages-list">
              <template v-if="pythonEnv.deployed">
                <div v-if="installedPackages.length > 0" class="package-items">
                  <div v-for="pkg in installedPackages" :key="pkg.name" class="package-item">
                    <div class="package-info">
                      <span class="package-name">{{ pkg.name }}</span>
                      <span class="package-version">{{ pkg.version }}</span>
                    </div>
                    <button class="btn-icon btn-uninstall" :disabled="isUninstalling === pkg.name" @click="confirmUninstallPackage(pkg.name)" :title="t('training.envDialog.uninstall')">
                      <Loader2 v-if="isUninstalling === pkg.name" :size="14" class="animate-spin" />
                      <X v-else :size="14" />
                    </button>
                  </div>
                </div>
                <div v-else class="packages-empty">{{ t('training.envDialog.noPackages') }}</div>
              </template>
              <div v-else class="packages-empty">{{ t('training.envDialog.deployFirst') }}</div>
            </div>
          </div>
        </div>
        <div class="modal-footer"><button class="btn" @click="showEnvDialog = false">{{ t('training.envDialog.close') }}</button></div>
      </div>
    </div>

    <!-- 卸载环境确认对话框 -->
    <div v-if="showUndeployConfirm" class="modal-overlay" @click.self="showUndeployConfirm = false">
      <div class="modal confirm-modal">
        <h2 class="modal-title">{{ t('training.envDialog.undeployConfirmTitle') }}</h2>
        <p class="confirm-text">{{ t('training.envDialog.undeployConfirmText') }}</p>
        <div class="modal-footer">
          <button class="btn" @click="showUndeployConfirm = false">{{ t('training.newTaskDialog.cancel') }}</button>
          <button class="btn btn-danger" :disabled="isUndeploying" @click="undeployEnvironment">
            <Loader2 v-if="isUndeploying" :size="14" class="animate-spin" />{{ t('training.envDialog.confirmUndeploy') }}
          </button>
        </div>
      </div>
    </div>

    <!-- 卸载依赖确认对话框 -->
    <div v-if="packageToUninstall" class="modal-overlay" @click.self="packageToUninstall = null">
      <div class="modal confirm-modal">
        <h2 class="modal-title">{{ t('training.envDialog.uninstallConfirmTitle') }}</h2>
        <p class="confirm-text">{{ t('training.envDialog.uninstallConfirmText', { name: packageToUninstall }) }}</p>
        <div class="modal-footer">
          <button class="btn" @click="packageToUninstall = null">{{ t('training.newTaskDialog.cancel') }}</button>
          <button class="btn btn-danger" :disabled="!!isUninstalling" @click="uninstallPackage">
            <Loader2 v-if="isUninstalling" :size="14" class="animate-spin" />{{ t('training.envDialog.confirmUninstall') }}
          </button>
        </div>
      </div>
    </div>

    <!-- 训练集管理对话框 -->
    <div v-if="showTrainsetDialog" class="modal-overlay" @click.self="showTrainsetDialog = false">
      <div class="modal trainset-modal">
        <h2 class="modal-title">{{ t('training.trainset.title') }}</h2>
        <div class="trainset-content">
          <!-- 左侧：项目树 -->
          <div class="trainset-tree">
            <div class="trainset-tree-header">{{ t('training.trainset.selectCategories') }}</div>
            <div class="trainset-tree-body">
              <div v-if="trainsetLoading" class="tree-loading">
                <Loader2 :size="20" class="animate-spin" />
              </div>
              <div v-else-if="trainsetProjects.length === 0" class="tree-empty">{{ t('training.trainset.noProjects') }}</div>
              <div v-else>
                <div v-for="project in trainsetProjects" :key="project.id" class="ts-tree-node">
                  <div class="ts-tree-row ts-tree-row--project" @click="toggleTrainsetProject(project.id)">
                    <component :is="trainsetExpandedProjects.has(project.id) ? ChevronDown : ChevronRight" :size="16" />
                    <Folder :size="16" />
                    <span>{{ project.name }}</span>
                  </div>
                  <div v-if="trainsetExpandedProjects.has(project.id)" class="ts-tree-children">
                    <div v-for="version in trainsetVersions[project.id] || []" :key="`${project.id}-v${version.version}`" class="ts-tree-node">
                      <div class="ts-tree-row ts-tree-row--version" @click="toggleTrainsetVersion(project.id, version.version)">
                        <component :is="trainsetExpandedVersions.has(`${project.id}-v${version.version}`) ? ChevronDown : ChevronRight" :size="16" />
                        <Tag :size="14" />
                        <span>v{{ version.version }}</span>
                        <span class="ts-tree-meta">{{ version.imageCount }} img</span>
                      </div>
                      <div v-if="trainsetExpandedVersions.has(`${project.id}-v${version.version}`)" class="ts-tree-children">
                        <div v-for="cat in trainsetCategories[`${project.id}-v${version.version}`] || []" :key="cat.id" :class="['ts-tree-row', 'ts-tree-row--category', { 'ts-tree-row--disabled': hasTrainsetCategoryConflict(project.id, version.version, cat.name) && !isTrainsetCategorySelected(project.id, version.version, cat.id) }]">
                          <input type="checkbox" 
                            :checked="isTrainsetCategorySelected(project.id, version.version, cat.id)" 
                            :disabled="hasTrainsetCategoryConflict(project.id, version.version, cat.name) && !isTrainsetCategorySelected(project.id, version.version, cat.id)"
                            @change="toggleTrainsetCategory(project.id, version.version, cat)" />
                          <Tag :size="14" :style="{ color: cat.color }" />
                          <span class="cat-name">{{ cat.name }}</span>
                          <span class="cat-type-tag">{{ getCategoryTypeLabel(cat.type) }}</span>
                          <span class="cat-stats">({{ cat.imageCount || 0 }} {{ t('training.trainset.images') }}, {{ cat.annotationCount || 0 }} {{ t('training.trainset.annotations') }})</span>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
            <!-- 选中的类别统计 -->
            <div v-if="trainsetSelectedCategories.length > 0" class="trainset-selected">
              <span>{{ t('training.trainset.selected', { count: trainsetSelectedCategories.length }) }}</span>
              <span class="trainset-stats-detail">
                {{ trainsetStats.totalImages }} {{ t('training.trainset.images') }} / {{ trainsetStats.totalAnnotations }} {{ t('training.trainset.annotations') }}
              </span>
            </div>
          </div>
          <!-- 中间：训练集信息编辑 -->
          <div class="trainset-editor">
            <div class="trainset-editor-header">{{ t('training.trainset.trainsetInfo') }}</div>
            <div class="trainset-editor-body">
              <div class="form-row">
                <label class="form-label">{{ t('training.trainset.trainsetName') }}</label>
                <input type="text" v-model="trainsetName" :placeholder="t('training.trainset.namePlaceholder')" class="input" />
              </div>
              <div class="form-row">
                <label class="form-label">{{ t('training.trainset.categoriesPreview') }}</label>
                <div class="categories-preview">
                  <div v-if="trainsetSelectedCategories.length === 0" class="preview-empty">{{ t('training.trainset.noCategoriesSelected') }}</div>
                  <div v-else class="preview-tags">
                    <span v-for="cat in trainsetSelectedCategories" :key="`${cat.projectId}-${cat.version}-${cat.categoryId}`" class="preview-tag" :title="`${cat.projectName || cat.projectId} v${cat.version}`">
                      {{ cat.categoryName }}
                      <span class="tag-source">{{ cat.projectName || cat.projectId }} v{{ cat.version }}</span>
                      <button class="tag-remove" @click="removeTrainsetCategory(cat)"><X :size="12" /></button>
                    </span>
                  </div>
                </div>
              </div>
            </div>
            <div class="trainset-editor-actions">
              <button class="btn btn-primary" :disabled="!canSaveTrainset || isSavingTrainset" @click="saveTrainset">
                <Loader2 v-if="isSavingTrainset" :size="14" class="animate-spin" />
                {{ editingTrainsetId ? t('training.trainset.update') : t('training.trainset.save') }}
              </button>
              <button v-if="editingTrainsetId" class="btn" @click="cancelEditTrainset">{{ t('training.trainset.cancelEdit') }}</button>
            </div>
          </div>
          <!-- 右侧：已保存的训练集列表 -->
          <div class="trainset-list">
            <div class="trainset-list-header">{{ t('training.trainset.savedTrainsets') }}</div>
            <div class="trainset-list-body">
              <div v-if="savedTrainsets.length === 0" class="list-empty">{{ t('training.trainset.noSavedTrainsets') }}</div>
              <div v-else class="trainset-items">
                <div v-for="ts in savedTrainsets" :key="ts.id" :class="['trainset-item', { 'trainset-item--active': editingTrainsetId === ts.id }]">
                  <div class="trainset-item-info" @click="loadTrainsetForEdit(ts)">
                    <span class="trainset-item-name">{{ ts.name }}</span>
                    <span class="trainset-item-meta">{{ ts.categories.length }} {{ t('training.trainset.categoriesCount') }}</span>
                  </div>
                  <div class="trainset-item-actions">
                    <button class="btn-icon" @click="loadTrainsetForEdit(ts)" :title="t('training.trainset.edit')"><Pencil :size="14" /></button>
                    <button class="btn-icon" @click="confirmDeleteTrainset(ts)" :title="t('training.trainset.delete')"><Trash2 :size="14" /></button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
        <div class="modal-footer">
          <button class="btn" @click="closeTrainsetDialog">{{ t('training.trainset.close') }}</button>
        </div>
      </div>
    </div>

    <!-- 删除训练集确认对话框 -->
    <div v-if="trainsetToDelete" class="modal-overlay" @click.self="trainsetToDelete = null">
      <div class="modal confirm-modal">
        <h2 class="modal-title">{{ t('training.trainset.deleteConfirmTitle') }}</h2>
        <p class="confirm-text">{{ t('training.trainset.deleteConfirmText', { name: trainsetToDelete.name }) }}</p>
        <div class="modal-footer">
          <button class="btn" @click="trainsetToDelete = null">{{ t('training.newTaskDialog.cancel') }}</button>
          <button class="btn btn-danger" @click="deleteTrainset">{{ t('training.trainset.confirmDelete') }}</button>
        </div>
      </div>
    </div>

    <!-- 新建训练任务对话框 -->
    <div v-if="showNewTaskDialog" class="modal-overlay" @click.self="showNewTaskDialog = false">
      <div class="modal new-task-modal-v2">
        <h2 class="modal-title">{{ t('training.newTaskDialog.title') }}</h2>
        <div class="new-task-form">
          <!-- 任务名称 -->
          <div class="form-row">
            <label class="form-label">{{ t('training.newTaskDialog.taskName') }}</label>
            <input type="text" v-model="newTaskForm.name" :placeholder="t('training.newTaskDialog.taskNamePlaceholder')" class="input" />
          </div>
          
          <!-- 数据来源 -->
          <div class="form-row">
            <label class="form-label">{{ t('training.newTaskDialog.datasetSource') }}</label>
            <div class="source-cards">
              <div :class="['source-card', { active: newTaskForm.sourceType === 'trainset' }]" @click="newTaskForm.sourceType = 'trainset'">
                <Database :size="20" /><span>{{ t('training.newTaskDialog.fromTrainset') }}</span>
              </div>
              <div :class="['source-card', { active: newTaskForm.sourceType === 'directory' }]" @click="newTaskForm.sourceType = 'directory'">
                <FolderOpen :size="20" /><span>{{ t('training.newTaskDialog.fromDirectory') }}</span>
              </div>
            </div>
          </div>
          
          <!-- 从训练集选择 -->
          <template v-if="newTaskForm.sourceType === 'trainset'">
            <div class="form-row">
              <label class="form-label">{{ t('training.newTaskDialog.selectTrainset') }}</label>
              <select v-model="newTaskForm.trainsetId" class="select">
                <option value="">{{ t('training.newTaskDialog.selectTrainsetPlaceholder') }}</option>
                <option v-for="ts in savedTrainsets" :key="ts.id" :value="ts.id">{{ ts.name }} ({{ ts.categories.length }} {{ t('training.trainset.categoriesCount') }})</option>
              </select>
            </div>
          </template>
          
          <!-- 从目录选择 -->
          <template v-else>
            <div class="form-row">
              <label class="form-label">{{ t('training.newTaskDialog.datasetPath') }}</label>
              <div class="path-input-row">
                <input type="text" v-model="newTaskForm.datasetPath" :placeholder="t('training.newTaskDialog.pathPlaceholder')" readonly class="input" />
                <button class="btn btn-browse" @click="browseDatasetPath">{{ t('training.newTaskDialog.browse') }}</button>
              </div>
            </div>
            <div v-if="newTaskForm.datasetPath" class="form-row">
              <label class="form-label">{{ t('training.newTaskDialog.detectedFormat') }}</label>
              <div class="detected-format">
                <Loader2 v-if="isDetectingFormat" :size="16" class="animate-spin" />
                <span v-else-if="detectedFormat" class="format-tag">{{ detectedFormat.toUpperCase() }}</span>
                <span v-else class="text-warning">{{ t('training.newTaskDialog.formatUnknown') }}</span>
              </div>
            </div>
          </template>
          
          <!-- 数据集划分 -->
          <div class="form-row">
            <label class="form-label">{{ t('training.newTaskDialog.trainValSplit') }}</label>
            <div class="split-row">
              <span class="split-label">Train</span>
              <input type="range" v-model.number="newTaskForm.trainRatio" min="50" max="95" step="5" class="split-slider" />
              <span class="split-value">{{ newTaskForm.trainRatio }}%</span>
              <span class="split-label">Val</span>
              <span class="split-value">{{ 100 - newTaskForm.trainRatio }}%</span>
            </div>
          </div>
          
          <!-- 训练类型 -->
          <div class="form-row">
            <label class="form-label">{{ t('training.newTaskDialog.trainType') }}</label>
            <select v-model="newTaskForm.trainType" class="select">
              <option v-for="opt in trainTypeOptions" :key="opt.value" :value="opt.value">{{ t(opt.label) }}</option>
            </select>
          </div>
          
          <!-- 训练插件 -->
          <div class="form-row">
            <label class="form-label">{{ t('training.newTaskDialog.trainPlugin') }}</label>
            <div class="plugin-select-row">
              <select v-model="newTaskForm.pluginId" class="select" :disabled="isLoadingPlugins || filteredPlugins.length === 0">
                <option value="">{{ isLoadingPlugins ? t('training.newTaskDialog.loadingPlugins') : t('training.newTaskDialog.selectPlugin') }}</option>
                <option v-for="p in filteredPlugins" :key="p.id" :value="p.id">{{ getLocalizedText(p.name) }}</option>
              </select>
              <Loader2 v-if="isLoadingPlugins" :size="16" class="animate-spin" />
            </div>
            <div v-if="filteredPlugins.length === 0 && !isLoadingPlugins" class="form-hint text-warning">
              {{ t('training.newTaskDialog.noPluginsForType') }}
            </div>
          </div>
          
          <!-- 依赖检查 -->
          <div v-if="selectedPlugin" class="form-row">
            <label class="form-label">{{ t('training.newTaskDialog.dependencies') }}</label>
            <div class="deps-status">
              <template v-if="isCheckingDeps">
                <Loader2 :size="14" class="animate-spin" />
                <span>{{ t('training.newTaskDialog.checkingDeps') }}</span>
              </template>
              <template v-else-if="!selectedPluginDeps">
                <AlertTriangle :size="14" class="text-warning" />
                <span class="text-warning">{{ t('training.newTaskDialog.checkingDeps') }}</span>
              </template>
              <template v-else-if="!selectedPluginDeps.deployed">
                <AlertTriangle :size="14" class="text-warning" />
                <span class="text-warning">{{ t('training.newTaskDialog.envNotDeployed') }}</span>
              </template>
              <template v-else-if="selectedPluginDeps.missing?.length > 0 || selectedPluginDeps.mismatched?.length > 0">
                <AlertCircle :size="14" class="text-error" />
                <span class="text-error">{{ t('training.newTaskDialog.depsMissing', { count: (selectedPluginDeps.missing?.length || 0) + (selectedPluginDeps.mismatched?.length || 0) }) }}</span>
                <button class="btn btn-small" @click="installMissingDeps" :disabled="isInstallingDeps">
                  <Loader2 v-if="isInstallingDeps" :size="12" class="animate-spin" />
                  {{ t('training.newTaskDialog.installDeps') }}
                </button>
              </template>
              <template v-else-if="selectedPluginDeps.torchInstalled && selectedPluginDeps.torchCudaAvailable === false">
                <AlertTriangle :size="14" class="text-warning" />
                <span class="text-warning">{{ t('training.newTaskDialog.torchCpuOnly') }}</span>
                <button class="btn btn-small" @click="reinstallTorchCuda" :disabled="isInstallingDeps">
                  <Loader2 v-if="isInstallingDeps" :size="12" class="animate-spin" />
                  {{ t('training.newTaskDialog.reinstallTorchCuda') }}
                </button>
              </template>
              <template v-else>
                <CheckCircle :size="14" class="text-success" />
                <span class="text-success">{{ t('training.newTaskDialog.depsOk') }}</span>
              </template>
            </div>
          </div>
          
          <!-- 模型选择 -->
          <div v-if="selectedPlugin" class="form-row">
            <label class="form-label">{{ t('training.newTaskDialog.model') }}</label>
            <div class="model-select-row">
              <!-- 左：模型选择，占 4 份 -->
              <select v-model="newTaskForm.modelId" class="select select-model">
                <option value="">{{ t('training.newTaskDialog.selectModel') }}</option>
                <option v-for="m in filteredModels" :key="m.id" :value="m.id">{{ m.name }}</option>
              </select>
              <!-- 中：模型变体，占 4 份（如果有） -->
              <select v-if="selectedModel?.variants?.length" v-model="newTaskForm.modelVariant" class="select select-variant">
                <option v-for="v in selectedModel.variants" :key="v" :value="v">{{ v.toUpperCase() }}</option>
              </select>
              <!-- 右：状态 + 操作按钮，占 2 份 -->
              <div class="model-actions">
                <template v-if="modelDownloadStatus">
                  <span v-if="modelDownloadStatus.status === 'completed'" class="model-status model-status--ok">
                    <CheckCircle :size="14" />
                  </span>
                  <button v-else class="btn btn-small btn-download" @click="downloadModel" :disabled="isDownloadingModel">
                    <Download :size="14" />
                    {{ t('training.newTaskDialog.downloadModel') }}
                  </button>
                </template>
                <button class="btn-icon" @click="openModelsDir" :title="t('training.newTaskDialog.openModelsDir')">
                  <FolderOpen :size="14" />
                </button>
                <button class="btn-icon" @click="checkModelStatus" :title="t('training.newTaskDialog.refreshModelStatus')">
                  <RefreshCw :size="14" />
                </button>
              </div>
            </div>
          </div>
          
          <!-- 超参数 -->
          <div class="form-row">
            <label class="form-label">{{ t('training.newTaskDialog.hyperparams') }}</label>
          </div>
          <div class="hyperparams-grid">
            <div class="param-item">
              <label>{{ t('training.newTaskDialog.epochs') }}</label>
              <input type="number" v-model.number="newTaskForm.epochs" min="1" max="1000" class="input" />
            </div>
            <div class="param-item">
              <label>{{ t('training.newTaskDialog.batchSize') }}</label>
              <input type="number" v-model.number="newTaskForm.batch" min="1" max="128" class="input" />
            </div>
            <div class="param-item">
              <label>{{ t('training.newTaskDialog.imageSize') }}</label>
              <select v-model.number="newTaskForm.imgsz" class="select">
                <option v-for="opt in imageSizeOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
              </select>
            </div>
            <!-- 插件自定义参数 -->
            <template v-for="param in pluginParamsSchema" :key="param.key">
              <div class="param-item" :title="param.description">
                <label>{{ param.title }}</label>
                <!-- 枚举选择 -->
                <select v-if="param.enum" v-model="pluginParams[param.key]" class="select">
                  <option v-for="opt in param.enum" :key="opt" :value="opt">{{ opt }}</option>
                </select>
                <!-- 布尔开关 -->
                <label v-else-if="param.type === 'boolean'" class="switch">
                  <input type="checkbox" v-model="pluginParams[param.key]" />
                  <span class="slider"></span>
                </label>
                <!-- 数值输入 -->
                <input 
                  v-else-if="param.type === 'number' || param.type === 'integer'"
                  type="number" 
                  v-model.number="pluginParams[param.key]" 
                  :min="param.minimum" 
                  :max="param.maximum"
                  :step="param.type === 'integer' ? 1 : 0.001"
                  class="input" 
                />
                <!-- 字符串输入 -->
                <input v-else type="text" v-model="pluginParams[param.key]" class="input" />
              </div>
            </template>
          </div>
        </div>
        <div class="modal-footer">
          <button class="btn" @click="showNewTaskDialog = false">{{ t('training.newTaskDialog.cancel') }}</button>
          <button class="btn btn-primary" :disabled="!canStartTraining || isStartingTask" @click="startTraining">
            <Loader2 v-if="isStartingTask" :size="16" class="animate-spin" />
            {{ isStartingTask ? t('training.newTaskDialog.starting') : t('training.newTaskDialog.start') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick, watch, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { 
  Cpu, MemoryStick, Monitor, HelpCircle, FolderPlus, Plus, Puzzle,
  Loader2, Clock, X, Check, MoreHorizontal, FileText, Copy, Trash2,
  FolderOpen, History, ChevronDown, ChevronRight, Folder, Tag, Pencil,
  Database, AlertTriangle, AlertCircle, CheckCircle, Download, RefreshCw
} from 'lucide-vue-next'
import { useRouter } from 'vue-router'
import notification from '../utils/notification'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'

const { t, locale } = useI18n()
const router = useRouter()
const API_BASE = 'http://127.0.0.1:18080'

// 获取本地化文本（支持字符串或国际化对象）
const getLocalizedText = (text: string | Record<string, string> | undefined): string => {
  if (!text) return ''
  if (typeof text === 'string') return text
  return text[locale.value] || text['zh-CN'] || text['en-US'] || Object.values(text)[0] || ''
}

// ============ 系统资源状态 ============
const systemStats = ref({ cpuUsage: 0, cpuCores: 0, memUsage: 0, memUsed: 0, memTotal: 0, gpuAvailable: false, gpuUsage: 0, vramUsed: 0, vramTotal: 0 })

// ============ 训练插件就绪状态 ============
const trainingPluginReady = computed(() => {
  // 有训练插件且依赖已满足
  if (trainingPlugins.value.length === 0) return false
  return trainingPluginDepsReady.value
})

// 检查训练插件依赖是否就绪
const checkTrainingPluginDeps = async () => {
  if (trainingPlugins.value.length === 0) {
    trainingPluginDepsReady.value = false
    return
  }
  
  // 检查第一个训练插件的依赖状态
  const plugin = trainingPlugins.value[0]
  if (!plugin) {
    trainingPluginDepsReady.value = false
    return
  }
  try {
    const res = await fetch(`http://localhost:18080/api/python/env-status?pluginId=${encodeURIComponent(plugin.id)}`)
    if (res.ok) {
      const data = await res.json()
      // 检查是否有虚拟环境且所有依赖都已安装
      if (!data.hasVenv) {
        trainingPluginDepsReady.value = false
        return
      }
      const deps = data.dependencies || []
      const missingDeps = deps.filter((d: any) => d.required && !d.installed)
      trainingPluginDepsReady.value = missingDeps.length === 0
    } else {
      trainingPluginDepsReady.value = false
    }
  } catch (e) {
    console.error('Check training plugin deps error:', e)
    trainingPluginDepsReady.value = false
  }
}

const navigateToPythonEnv = () => {
  router.push('/python-env')
}

// ============ Python 环境状态 ============
const pythonEnv = ref({ deployed: false, pythonVersion: '', venvPath: '' })
const isPythonVersionWarning = computed(() => {
  if (!pythonEnv.value.pythonVersion) return false
  const v = pythonEnv.value.pythonVersion.split('.')
  const major = parseInt(v[0] || '0'), minor = parseInt(v[1] || '0')
  return major !== 3 || minor < 8 || minor > 12
})
const showEnvDialog = ref(false)
const isDeploying = ref(false)
const isUndeploying = ref(false)
const showUndeployConfirm = ref(false)
const isInstalling = ref(false)
const isUninstalling = ref<string | null>(null)
const packageToInstall = ref('')
const packageToUninstall = ref<string | null>(null)
const installedPackages = ref<Array<{ name: string; version: string }>>([])

// 依赖源设置
const usePypiMirror = ref(true)
const pypiMirrorUrl = ref('https://pypi.tuna.tsinghua.edu.cn/simple')

// ============ 训练任务 ============
interface TrainingTask {
  id: string; name: string; status: 'running' | 'completed' | 'error' | 'pending'
  currentEpoch: number; totalEpochs: number; batchProgress: number
  progress?: { epoch?: number; totalEpochs?: number; progress?: number; metrics?: Record<string, number> }
  error?: string
  logs: string[]
  lastLogIndex: number // 跟踪已显示的日志
  metrics: { boxLoss?: number; clsLoss?: number; trainLoss?: number; mAP50?: number; mAP5095?: number; valLoss?: number }
}
const trainingTasks = ref<TrainingTask[]>([])
const openMenuTaskId = ref<string | null>(null)
const toggleTaskMenu = (id: string) => { openMenuTaskId.value = openMenuTaskId.value === id ? null : id }

const handleTaskAction = async (action: string, task: TrainingTask) => {
  openMenuTaskId.value = null
  
  if (action === 'stop') {
    // 停止正在运行的任务
    try {
      const res = await fetch(`${API_BASE}/api/training/stop`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ taskId: task.id })
      })
      if (res.ok) {
        task.status = 'error'
        notification.success(t('training.tasks.stopSuccess'))
      } else {
        notification.error(t('training.tasks.stopFailed'))
      }
    } catch {
      notification.error(t('training.tasks.stopFailed'))
    }
  } else if (action === 'delete') {
    // 删除任务（从列表中移除，可选删除输出目录）
    try {
      const res = await fetch(`${API_BASE}/api/training/delete`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ taskId: task.id, deleteOutput: true })
      })
      if (res.ok) {
        trainingTasks.value = trainingTasks.value.filter(t => t.id !== task.id)
        notification.success(t('training.tasks.deleteSuccess'))
      } else {
        notification.error(t('training.tasks.deleteFailed'))
      }
    } catch {
      notification.error(t('training.tasks.deleteFailed'))
    }
  } else if (action === 'retrain') {
    // 重新训练：使用相同参数新建任务
    // 先删除当前失败的任务
    try {
      await fetch(`${API_BASE}/api/training/delete`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ taskId: task.id, deleteOutput: true })
      })
      trainingTasks.value = trainingTasks.value.filter(t => t.id !== task.id)
    } catch { /* ignore */ }
    
    // 打开新建任务对话框
    showNewTaskDialog.value = true
    notification.info(t('training.tasks.retrainHint'))
  }
}

// 加载训练任务列表
const loadTrainingTasks = async () => {
  try {
    const res = await fetch(`${API_BASE}/api/training/tasks`)
    if (res.ok) {
      const data = await res.json()
      const oldTasks = new Map(trainingTasks.value.map(t => [t.id, t]))
      // 只保留未完成的任务（running/pending/error 等），已完成的走历史记录
      trainingTasks.value = (data.tasks || [])
        .filter((t: any) => t.status !== 'completed')
        .map((t: any) => {
        const oldTask = oldTasks.get(t.id)
        const lastLogIndex = oldTask?.lastLogIndex || 0
        const logs = t.logs || []
        
        // 将新日志追加到虚拟终端
        if (logs.length > lastLogIndex) {
          for (let i = lastLogIndex; i < logs.length; i++) {
            appendLog(logs[i], 'info')
          }
        }
        
        return {
          id: t.id,
          name: t.name,
          status: t.status,
          currentEpoch: t.progress?.epoch || 0,
          totalEpochs: t.progress?.totalEpochs || t.params?.epochs || 0,
          batchProgress: (t.progress?.progress || 0) * 100,
          progress: t.progress,
          error: t.error,
          logs: logs,
          lastLogIndex: logs.length,
          metrics: t.progress?.metrics || {}
        }
      })
    }
  } catch (e) { console.error('Load tasks failed:', e) }
}

// ============ 训练 WebSocket ============
let trainingWS: WebSocket | null = null
let wsReconnectTimer: number | null = null

const connectTrainingWS = () => {
  if (trainingWS?.readyState === WebSocket.OPEN) return
  
  // 打包后 file:// 协议下 hostname 为空，使用 localhost
  const host = window.location.hostname || 'localhost'
  const wsUrl = `ws://${host}:18080/api/training/ws`
  trainingWS = new WebSocket(wsUrl)
  
  trainingWS.onopen = () => {
    console.log('[TrainingWS] Connected')
    if (wsReconnectTimer) { clearTimeout(wsReconnectTimer); wsReconnectTimer = null }
  }
  
  trainingWS.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data)
      handleWSMessage(msg)
    } catch (e) { console.error('[TrainingWS] Parse error:', e) }
  }
  
  trainingWS.onclose = () => {
    console.log('[TrainingWS] Disconnected, reconnecting in 3s...')
    trainingWS = null
    wsReconnectTimer = window.setTimeout(connectTrainingWS, 3000)
  }
  
  trainingWS.onerror = (e) => {
    console.error('[TrainingWS] Error:', e)
  }
}

const handleWSMessage = (msg: any) => {
  const { type, taskId } = msg
  
  if (type === 'init' && msg.tasks) {
    // 初始化所有任务（仅保留未完成的任务）
    trainingTasks.value = msg.tasks
      .filter((t: any) => t.status !== 'completed')
      .map((t: any) => mapTask(t))
    // 显示已有日志
    for (const task of trainingTasks.value) {
      for (const log of task.logs) {
        appendLog(log, 'info')
      }
      task.lastLogIndex = task.logs.length
    }
    return
  }
  
  if (type === 'resources' && msg.stats) {
    // 更新系统资源状态
    systemStats.value = {
      cpuUsage: msg.stats.cpuUsage || 0,
      cpuCores: msg.stats.cpuCores || 0,
      memUsage: msg.stats.memUsage || 0,
      memUsed: msg.stats.memUsed || 0,
      memTotal: msg.stats.memTotal || 0,
      gpuAvailable: msg.stats.gpuAvailable || false,
      gpuUsage: msg.stats.gpuUsage || 0,
      vramUsed: msg.stats.vramUsed || 0,
      vramTotal: msg.stats.vramTotal || 0
    }
    return
  }
  
  if (!taskId) return
  const idx = trainingTasks.value.findIndex(t => t.id === taskId)
  if (idx < 0) return
  
  const task = trainingTasks.value[idx]!
  
  switch (type) {
    case 'progress':
    case 'epoch_end': {
      // 更新进度和指标
      const p = msg.progress
      if (p) {
        task.currentEpoch = p.epoch ?? task.currentEpoch
        task.totalEpochs = p.totalEpochs ?? task.totalEpochs
        task.batchProgress = (p.progress ?? 0) * 100
        // 更新指标：从 metrics 对象和顶层都读取
        if (p.metrics) {
          task.metrics = { ...task.metrics, ...p.metrics }
        }
        // 验证指标在顶层（后端合并后的位置），始终检查
        if (p.boxLoss !== undefined) task.metrics.boxLoss = p.boxLoss
        if (p.clsLoss !== undefined) task.metrics.clsLoss = p.clsLoss
        if (p.trainLoss !== undefined) task.metrics.trainLoss = p.trainLoss
        if (p.mAP50 !== undefined) task.metrics.mAP50 = p.mAP50
        if (p.mAP5095 !== undefined) task.metrics.mAP5095 = p.mAP5095
        if (p.valLoss !== undefined) task.metrics.valLoss = p.valLoss
      }
      if (msg.status) task.status = msg.status
      break
    }
    case 'log':
      // 追加单条日志
      if (msg.log) {
        appendLog(msg.log, 'info')
        task.logs.push(msg.log)
        task.lastLogIndex = task.logs.length
      }
      break
    case 'done':
      task.status = 'completed'
      // 将已完成的任务添加到历史记录（避免重复）
      if (!trainingHistory.value.find(r => r.id === task.id)) {
        trainingHistory.value.unshift({
          id: task.id,
          name: task.name,
          outputPath: msg.outputPath || '',
          completedAt: new Date().toLocaleString()
        })
      }
      // 从活跃任务列表中移除
      trainingTasks.value = trainingTasks.value.filter(t => t.id !== taskId)
      notification.success(t('training.tasks.completedNotify', { name: task.name }))
      break
    case 'error':
      task.status = 'error'
      if (msg.error) task.error = msg.error
      break
  }
}

const mapTask = (t: any): TrainingTask => {
  const p = t.progress || {}
  return {
    id: t.id,
    name: t.name,
    status: t.status,
    currentEpoch: p.epoch || 0,
    totalEpochs: p.totalEpochs || t.params?.epochs || 0,
    batchProgress: (p.progress || 0) * 100,
    progress: p,
    error: t.error,
    logs: t.logs || [],
    lastLogIndex: 0,
    metrics: {
      boxLoss: p.boxLoss ?? p.metrics?.boxLoss,
      clsLoss: p.clsLoss ?? p.metrics?.clsLoss,
      trainLoss: p.trainLoss ?? p.metrics?.trainLoss,
      mAP50: p.mAP50 ?? p.metrics?.mAP50,
      mAP5095: p.mAP5095 ?? p.metrics?.mAP5095,
      valLoss: p.valLoss ?? p.metrics?.valLoss
    }
  }
}

const disconnectTrainingWS = () => {
  if (wsReconnectTimer) { clearTimeout(wsReconnectTimer); wsReconnectTimer = null }
  if (trainingWS) { trainingWS.close(); trainingWS = null }
}

// 兼容旧的函数名
const startTasksRefresh = () => connectTrainingWS()
const stopTasksRefresh = () => disconnectTrainingWS()

// ============ 训练集管理 ============
interface TrainsetProject { id: string; name: string }
interface TrainsetVersion { version: number; imageCount: number; categoryCount: number }
interface TrainsetCategory { id: number; name: string; type: string; color: string; annotationCount: number; imageCount: number }
interface SelectedTrainsetCategory { projectId: string; projectName?: string; version: number; categoryId: number; categoryName: string; categoryType: string; annotationCount: number; imageCount: number }
interface SavedTrainset { id: string; name: string; categories: SelectedTrainsetCategory[]; createdAt: string; updatedAt: string }

const showTrainsetDialog = ref(false)
const trainsetLoading = ref(false)
const trainsetProjects = ref<TrainsetProject[]>([])
const trainsetVersions = ref<Record<string, TrainsetVersion[]>>({})
const trainsetCategories = ref<Record<string, TrainsetCategory[]>>({})
const trainsetExpandedProjects = ref<Set<string>>(new Set())
const trainsetExpandedVersions = ref<Set<string>>(new Set())
const trainsetSelectedCategories = ref<SelectedTrainsetCategory[]>([])
const trainsetName = ref('')
const savedTrainsets = ref<SavedTrainset[]>([])
const editingTrainsetId = ref<string | null>(null)
const isSavingTrainset = ref(false)
const trainsetToDelete = ref<SavedTrainset | null>(null)

const canSaveTrainset = computed(() => trainsetName.value.trim() && trainsetSelectedCategories.value.length > 0)

// 计算已选类别的统计信息
const trainsetStats = computed(() => {
  let totalImages = 0
  let totalAnnotations = 0
  for (const cat of trainsetSelectedCategories.value) {
    totalAnnotations += cat.annotationCount || 0
    totalImages += cat.imageCount || 0
  }
  return { totalImages, totalAnnotations }
})

const getCategoryTypeLabel = (type: string) => {
  switch (type) {
    case 'rectangle': return t('training.trainset.typeRectangle')
    case 'polygon': return t('training.trainset.typePolygon')
    case 'keypoint': return t('training.trainset.typeKeypoint')
    default: return type
  }
}

const loadTrainsetProjects = async () => {
  trainsetLoading.value = true
  try {
    const res = await fetch(`${API_BASE}/api/projects`)
    if (res.ok) {
      const data = await res.json()
      trainsetProjects.value = Array.isArray(data) ? data : (data.projects || [])
    }
  } catch { trainsetProjects.value = [] }
  trainsetLoading.value = false
}

const loadTrainsetVersions = async (projectId: string) => {
  try {
    const res = await fetch(`${API_BASE}/api/dataset-versions?projectId=${projectId}`)
    if (res.ok) { trainsetVersions.value[projectId] = (await res.json()).versions || [] }
  } catch { trainsetVersions.value[projectId] = [] }
}

const loadTrainsetCategories = async (projectId: string, version: number) => {
  const key = `${projectId}-v${version}`
  try {
    const res = await fetch(`${API_BASE}/api/project-categories?projectId=${projectId}&version=${version}`)
    if (res.ok) {
      const data = await res.json()
      trainsetCategories.value[key] = Array.isArray(data) ? data : (data.items || data.categories || [])
    }
  } catch { trainsetCategories.value[key] = [] }
}

const toggleTrainsetProject = async (projectId: string) => {
  if (trainsetExpandedProjects.value.has(projectId)) {
    trainsetExpandedProjects.value.delete(projectId)
  } else {
    trainsetExpandedProjects.value.add(projectId)
    if (!trainsetVersions.value[projectId]) await loadTrainsetVersions(projectId)
  }
}

const toggleTrainsetVersion = async (projectId: string, version: number) => {
  const key = `${projectId}-v${version}`
  if (trainsetExpandedVersions.value.has(key)) {
    trainsetExpandedVersions.value.delete(key)
  } else {
    trainsetExpandedVersions.value.add(key)
    if (!trainsetCategories.value[key]) await loadTrainsetCategories(projectId, version)
  }
}

const isTrainsetCategorySelected = (projectId: string, version: number, categoryId: number) => {
  return trainsetSelectedCategories.value.some(c => c.projectId === projectId && c.version === version && c.categoryId === categoryId)
}

// 检查类别是否有冲突（同项目不同版本的相同类别名）
const hasTrainsetCategoryConflict = (projectId: string, version: number, categoryName: string) => {
  return trainsetSelectedCategories.value.some(c => c.projectId === projectId && c.categoryName === categoryName && c.version !== version)
}

const toggleTrainsetCategory = (projectId: string, version: number, cat: TrainsetCategory) => {
  const idx = trainsetSelectedCategories.value.findIndex(c => c.projectId === projectId && c.version === version && c.categoryId === cat.id)
  if (idx >= 0) {
    trainsetSelectedCategories.value.splice(idx, 1)
  } else {
    // 检查冲突：同一项目不同版本的相同类别名
    if (hasTrainsetCategoryConflict(projectId, version, cat.name)) {
      notification.warning(t('training.trainset.conflictWarning'))
      return
    }
    // 获取项目名称
    const project = trainsetProjects.value.find(p => p.id === projectId)
    trainsetSelectedCategories.value.push({ 
      projectId, projectName: project?.name, version, categoryId: cat.id, categoryName: cat.name,
      categoryType: cat.type, annotationCount: cat.annotationCount || 0, imageCount: cat.imageCount || 0
    })
  }
}

const removeTrainsetCategory = (cat: SelectedTrainsetCategory) => {
  const idx = trainsetSelectedCategories.value.findIndex(c => c.projectId === cat.projectId && c.version === cat.version && c.categoryId === cat.categoryId)
  if (idx >= 0) trainsetSelectedCategories.value.splice(idx, 1)
}

const loadSavedTrainsets = async () => {
  try {
    const res = await fetch(`${API_BASE}/api/trainsets`)
    if (res.ok) { savedTrainsets.value = (await res.json()).trainsets || [] }
  } catch { savedTrainsets.value = [] }
}

const saveTrainset = async () => {
  if (!canSaveTrainset.value) return
  isSavingTrainset.value = true
  try {
    const body = {
      id: editingTrainsetId.value || undefined,
      name: trainsetName.value.trim(),
      categories: trainsetSelectedCategories.value
    }
    const res = await fetch(`${API_BASE}/api/trainsets/save`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body)
    })
    if (res.ok) {
      notification.success(editingTrainsetId.value ? t('training.trainset.updateSuccess') : t('training.trainset.saveSuccess'))
      await loadSavedTrainsets()
      cancelEditTrainset()
    } else {
      notification.error(t('training.trainset.saveError'))
    }
  } catch { notification.error(t('training.trainset.saveError')) }
  isSavingTrainset.value = false
}

const loadTrainsetForEdit = async (ts: SavedTrainset) => {
  editingTrainsetId.value = ts.id
  trainsetName.value = ts.name
  
  // 补充项目名称（后端可能没有保存）
  const categoriesWithNames = ts.categories.map(cat => {
    if (!cat.projectName) {
      const project = trainsetProjects.value.find(p => p.id === cat.projectId)
      return { ...cat, projectName: project?.name }
    }
    return cat
  })
  trainsetSelectedCategories.value = categoriesWithNames
  
  // 自动展开并加载训练集中包含的项目和版本
  for (const cat of ts.categories) {
    // 展开项目
    if (!trainsetExpandedProjects.value.has(cat.projectId)) {
      trainsetExpandedProjects.value.add(cat.projectId)
      if (!trainsetVersions.value[cat.projectId]) {
        await loadTrainsetVersions(cat.projectId)
      }
    }
    // 展开版本
    const versionKey = `${cat.projectId}-v${cat.version}`
    if (!trainsetExpandedVersions.value.has(versionKey)) {
      trainsetExpandedVersions.value.add(versionKey)
      if (!trainsetCategories.value[versionKey]) {
        await loadTrainsetCategories(cat.projectId, cat.version)
      }
    }
  }
}

const cancelEditTrainset = () => {
  editingTrainsetId.value = null
  trainsetName.value = ''
  trainsetSelectedCategories.value = []
}

const confirmDeleteTrainset = (ts: SavedTrainset) => {
  trainsetToDelete.value = ts
}

const deleteTrainset = async () => {
  if (!trainsetToDelete.value) return
  try {
    const res = await fetch(`${API_BASE}/api/trainsets/delete`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id: trainsetToDelete.value.id })
    })
    if (res.ok) {
      notification.success(t('training.trainset.deleteSuccess'))
      await loadSavedTrainsets()
      if (editingTrainsetId.value === trainsetToDelete.value.id) cancelEditTrainset()
    } else {
      notification.error(t('training.trainset.deleteError'))
    }
  } catch { notification.error(t('training.trainset.deleteError')) }
  trainsetToDelete.value = null
}

const closeTrainsetDialog = () => {
  showTrainsetDialog.value = false
  cancelEditTrainset()
}

watch(showTrainsetDialog, async (val) => {
  if (val) {
    await loadTrainsetProjects()
    await loadSavedTrainsets()
  }
})

// ============ 终端窗口 (xterm.js) ============
const terminalContainerRef = ref<HTMLElement | null>(null)
const terminal = shallowRef<Terminal | null>(null)
const fitAddon = shallowRef<FitAddon | null>(null)

// 终端主题配置
const terminalThemes = {
  dark: {
    background: '#0d0d0d',
    foreground: '#c8c8c8',
    cursor: '#c8c8c8',
    cursorAccent: '#0d0d0d',
    selectionBackground: '#3a3a3a',
    black: '#000000',
    red: '#e06c75',
    green: '#98c379',
    yellow: '#e5c07b',
    blue: '#61afef',
    magenta: '#c678dd',
    cyan: '#56b6c2',
    white: '#abb2bf',
    brightBlack: '#5c6370',
    brightRed: '#e06c75',
    brightGreen: '#98c379',
    brightYellow: '#e5c07b',
    brightBlue: '#61afef',
    brightMagenta: '#c678dd',
    brightCyan: '#56b6c2',
    brightWhite: '#ffffff'
  },
  light: {
    background: '#f5f5f5',
    foreground: '#1f1f1f',
    cursor: '#1f1f1f',
    cursorAccent: '#f5f5f5',
    selectionBackground: '#d0d0d0',
    black: '#1f1f1f',
    red: '#c41a16',
    green: '#007400',
    yellow: '#826b28',
    blue: '#0451a5',
    magenta: '#bc05bc',
    cyan: '#0598bc',
    white: '#808080',
    brightBlack: '#5f5f5f',
    brightRed: '#cd3131',
    brightGreen: '#14ce14',
    brightYellow: '#c4a500',
    brightBlue: '#0451a5',
    brightMagenta: '#bc05bc',
    brightCyan: '#0598bc',
    brightWhite: '#a0a0a0'
  }
}

const getCurrentTheme = () => document.documentElement.getAttribute('data-theme') === 'light' ? 'light' : 'dark'

const updateTerminalTheme = () => {
  if (!terminal.value) return
  const theme = getCurrentTheme()
  terminal.value.options.theme = terminalThemes[theme]
}

const initTerminal = () => {
  if (!terminalContainerRef.value || terminal.value) return
  
  const theme = getCurrentTheme()
  const term = new Terminal({
    theme: terminalThemes[theme],
    fontSize: 13,
    fontFamily: "'Consolas', 'Monaco', 'Courier New', monospace",
    cursorBlink: false,
    cursorStyle: 'underline',
    scrollback: 5000,
    convertEol: true
  })
  
  const fit = new FitAddon()
  term.loadAddon(fit)
  term.open(terminalContainerRef.value)
  fit.fit()
  
  terminal.value = term
  fitAddon.value = fit
  
  // 监听容器大小变化
  const resizeObserver = new ResizeObserver(() => {
    nextTick(() => fitAddon.value?.fit())
  })
  resizeObserver.observe(terminalContainerRef.value)
  
  // 监听主题变化
  const themeObserver = new MutationObserver(() => updateTerminalTheme())
  themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['data-theme'] })
}

const clearTerminal = () => {
  terminal.value?.clear()
}

const copyTerminalContent = async () => {
  if (!terminal.value) return
  // 获取终端所有行的文本内容
  const buffer = terminal.value.buffer.active
  const lines: string[] = []
  for (let i = 0; i < buffer.length; i++) {
    const line = buffer.getLine(i)
    if (line) lines.push(line.translateToString(true))
  }
  const text = lines.join('\n').trimEnd()
  if (!text) {
    notification.warning(t('training.logs.empty'))
    return
  }
  try {
    await navigator.clipboard.writeText(text)
    notification.success(t('training.logs.copied'))
  } catch {
    notification.error(t('training.logs.copyFailed'))
  }
}

const getTimeStr = () => { 
  const d = new Date()
  return `${d.getHours().toString().padStart(2,'0')}:${d.getMinutes().toString().padStart(2,'0')}:${d.getSeconds().toString().padStart(2,'0')}` 
}

// 写入原始输出（保留 ANSI 转义序列，用于 pip 进度条等）
const writeRaw = (text: string) => {
  if (!terminal.value) return
  terminal.value.write(text)
}

// 带时间戳的日志输出
const appendLog = (text: string, type: 'cmd' | 'info' | 'error' | 'raw' = 'info') => {
  if (!terminal.value) return
  
  if (type === 'raw') {
    // 原始输出，直接写入（保留进度条等）
    writeRaw(text)
    return
  }
  
  const time = `\x1b[90m[${getTimeStr()}]\x1b[0m ` // 灰色时间戳
  
  if (type === 'cmd') {
    terminal.value.writeln(time + `\x1b[36m${text}\x1b[0m`) // 青色命令
  } else if (type === 'error') {
    terminal.value.writeln(time + `\x1b[31m${text}\x1b[0m`) // 红色错误
  } else {
    // 普通日志，按行输出
    const lines = text.split('\n')
    for (const line of lines) {
      if (line.trim()) terminal.value.writeln(time + line)
    }
  }
}

// ============ 训练历史 ============
interface TrainingRecord { id: string; name: string; outputPath: string; completedAt: string }
const trainingHistory = ref<TrainingRecord[]>([])
const openHistoryMenuId = ref<string | null>(null)
const toggleHistoryMenu = (id: string) => { openHistoryMenuId.value = openHistoryMenuId.value === id ? null : id }

// 加载训练历史（从后端扫描输出目录）
const loadTrainingHistory = async () => {
  try {
    const res = await fetch(`${API_BASE}/api/training/history`)
    if (res.ok) {
      const data = await res.json()
      trainingHistory.value = (data.history || []).map((h: any) => ({
        id: h.id,
        name: h.name || h.id,
        outputPath: h.outputPath || '',
        completedAt: h.completedAt ? new Date(h.completedAt).toLocaleString() : ''
      }))
    }
  } catch (e) {
    console.error('Failed to load training history:', e)
  }
}

const openHistoryFolder = (path: string) => { const api = (window as any)?.electronAPI; if (api?.openPath) api.openPath(path) }

const handleHistoryAction = async (action: string, record: TrainingRecord) => {
  openHistoryMenuId.value = null
  if (action === 'open') {
    openHistoryFolder(record.outputPath)
  } else if (action === 'delete') {
    try {
      const res = await fetch(`${API_BASE}/api/training/delete`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ taskId: record.id, deleteOutput: true })
      })
      if (res.ok) {
        trainingHistory.value = trainingHistory.value.filter(r => r.id !== record.id)
        notification.success(t('training.history.deleteSuccess'))
      } else {
        notification.error(t('training.history.deleteFailed'))
      }
    } catch {
      notification.error(t('training.history.deleteFailed'))
    }
  }
}

// ============ 分割线拖拽 ============
const bottomHeight = ref(200)
let isResizing = false, startY = 0, startHeight = 0
const startResize = (e: MouseEvent) => {
  isResizing = true; startY = e.clientY; startHeight = bottomHeight.value
  document.addEventListener('mousemove', onResize); document.addEventListener('mouseup', stopResize)
  document.body.style.cursor = 'ns-resize'; document.body.style.userSelect = 'none'
}
const onResize = (e: MouseEvent) => {
  if (!isResizing) return
  const lp = document.querySelector('.left-panel') as HTMLElement
  if (!lp) return
  let h = startHeight + (startY - e.clientY)
  h = Math.max(100, Math.min(lp.clientHeight * 0.5, h))
  bottomHeight.value = h
}
const stopResize = () => { isResizing = false; document.removeEventListener('mousemove', onResize); document.removeEventListener('mouseup', stopResize); document.body.style.cursor = ''; document.body.style.userSelect = '' }

// ============ 新建训练任务 ============
const showNewTaskDialog = ref(false)
const isStartingTask = ref(false)
const isDetectingFormat = ref(false)
const detectedFormat = ref<string | null>(null)

// 训练类型选项
const trainTypeOptions = [
  { value: 'detection', label: 'training.newTaskDialog.trainTypes.detection' },
  { value: 'pose', label: 'training.newTaskDialog.trainTypes.pose' },
  { value: 'segmentation', label: 'training.newTaskDialog.trainTypes.segmentation' },
  { value: 'custom', label: 'training.newTaskDialog.trainTypes.custom' }
]

// 训练插件相关
interface TorchRequirements {
  packages: string[]
  indexUrl: string
  description?: string
}
interface TrainingPlugin {
  id: string; name: string | Record<string, string>; version: string; description: string | Record<string, string>
  training?: { trainTypes: string[]; dataFormat: string; framework?: string }
  models: PluginModel[]; python?: { minVersion: string; requirements: string[]; torchRequirements?: TorchRequirements }
  paramsSchema?: Record<string, any>; pluginPath: string; entry?: string
}
interface PluginModel {
  id: string; name: string; trainTypes: string[]; variants: string[]
  isStandard: boolean; downloadUrl?: string
}
interface DepsCheckResult {
  deployed: boolean; missing: string[]; installed: string[]; mismatched: string[]
  torchInstalled?: boolean
  torchCudaAvailable?: boolean
}

const trainingPlugins = ref<TrainingPlugin[]>([])
const isLoadingPlugins = ref(false)
const selectedPluginDeps = ref<DepsCheckResult | null>(null)
const isCheckingDeps = ref(false)
const isInstallingDeps = ref(false)

// 训练插件依赖就绪状态
const trainingPluginDepsReady = ref(false)

// 插件自定义参数
const pluginParams = ref<Record<string, any>>({})

// 模型下载状态
interface ModelDownloadStatus {
  modelId: string; status: 'downloading' | 'completed' | 'failed' | 'not_downloaded'
  progress: number; downloaded: number; total: number; error?: string; exists?: boolean
}
const modelDownloadStatus = ref<ModelDownloadStatus | null>(null)
const isDownloadingModel = ref(false)
const modelStatusPollTimer = ref<number | null>(null)

// 表单数据
const newTaskForm = ref({
  name: '',
  sourceType: 'trainset' as 'trainset' | 'directory',
  trainsetId: '',
  datasetPath: '',
  trainType: 'detection',
  pluginId: '',
  modelId: '',
  modelVariant: 'n',
  trainRatio: 80,
  epochs: 100,
  batch: 16,
  imgsz: 640
})

const generateDefaultTaskName = () => {
  const n = new Date()
  const p = (x: number) => x.toString().padStart(2, '0')
  return `${n.getFullYear()}-${p(n.getMonth() + 1)}-${p(n.getDate())}_${p(n.getHours())}:${p(n.getMinutes())}`
}

const imageSizeOptions = [
  { label: '320', value: 320 },
  { label: '640', value: 640 },
  { label: '1280', value: 1280 }
]

// 根据训练类型筛选插件
const filteredPlugins = computed(() => {
  const type = newTaskForm.value.trainType
  return trainingPlugins.value.filter(p => {
    // 从 training.trainTypes 获取支持的训练类型
    const types = p.training?.trainTypes || []
    if (type === 'custom') return types.includes('custom')
    return types.includes(type)
  })
})

// 当前选中的插件
const selectedPlugin = computed(() => {
  return trainingPlugins.value.find(p => p.id === newTaskForm.value.pluginId) || null
})

// 当前插件的参数 Schema
const pluginParamsSchema = computed(() => {
  const schema = selectedPlugin.value?.paramsSchema
  if (!schema || !schema.properties) return []
  return Object.entries(schema.properties).map(([key, prop]: [string, any]) => ({
    key,
    type: prop.type || 'string',
    title: prop.title || key,
    description: prop.description || '',
    default: prop.default,
    minimum: prop.minimum,
    maximum: prop.maximum,
    enum: prop.enum
  }))
})

// 根据训练类型筛选模型
const filteredModels = computed(() => {
  if (!selectedPlugin.value) return []
  const type = newTaskForm.value.trainType
  return selectedPlugin.value.models.filter(m => m.trainTypes.includes(type) || m.trainTypes.includes('custom'))
})

// 当前选中的模型
const selectedModel = computed(() => {
  if (!selectedPlugin.value) return null
  return selectedPlugin.value.models.find(m => m.id === newTaskForm.value.modelId) || null
})

// 当前模型的完整ID（含变体）
const fullModelId = computed(() => {
  if (!selectedModel.value) return ''
  const variant = newTaskForm.value.modelVariant
  // 根据模型ID和变体构建完整ID，如 yolov8 + n = yolov8n
  const baseId = selectedModel.value.id
  if (baseId.includes('-')) {
    // 如 yolov8-pose -> yolov8n-pose
    const parts = baseId.split('-')
    return `${parts[0]}${variant}-${parts.slice(1).join('-')}`
  }
  return `${baseId}${variant}`
})

// 加载训练插件列表
const loadTrainingPlugins = async () => {
  isLoadingPlugins.value = true
  try {
    const res = await fetch(`${API_BASE}/api/training/plugins`)
    if (res.ok) {
      const data = await res.json()
      console.log('[Training] Loaded plugins:', data.plugins)
      trainingPlugins.value = data.plugins || []
    }
  } catch (e) { console.error('Load plugins failed:', e) }
  isLoadingPlugins.value = false
}

// 检查插件依赖（使用虚拟环境状态 API）
const checkPluginDeps = async () => {
  const plugin = selectedPlugin.value
  if (!plugin) {
    selectedPluginDeps.value = null
    return
  }
  isCheckingDeps.value = true
  try {
    const res = await fetch(`${API_BASE}/api/python/env-status?pluginId=${encodeURIComponent(plugin.id)}`)
    if (res.ok) {
      const data = await res.json()
      // 转换为 DepsCheckResult 格式
      const missing: string[] = []
      const installed: string[] = []
      let torchInstalled = false
      
      for (const dep of data.dependencies || []) {
        if (dep.installed) {
          installed.push(dep.name)
          if (dep.name.toLowerCase().startsWith('torch')) {
            torchInstalled = true
          }
        } else {
          missing.push(dep.name)
        }
      }
      
      selectedPluginDeps.value = {
        deployed: data.hasVenv || false,
        missing,
        installed,
        mismatched: [],
        torchInstalled,
        torchCudaAvailable: torchInstalled // 暂时假设有 torch 就可用
      }
    }
  } catch { selectedPluginDeps.value = null }
  isCheckingDeps.value = false
}

// 运行 pip 安装并输出到终端
const runPipInstall = async (pkgs: string[], indexUrl?: string) => {
  const mirror = usePypiMirror.value ? pypiMirrorUrl.value : ''
  const body: Record<string, string> = { package: pkgs.join(' ') }
  if (indexUrl) body.indexUrl = indexUrl
  else if (mirror) body.mirror = mirror
  
  appendLog(`pip install ${pkgs.join(' ')}${indexUrl ? ` --index-url ${indexUrl}` : ''}`, 'cmd')
  
  const res = await fetch(`${API_BASE}/api/python/install`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body)
  })
  if (!res.ok) throw new Error('Install request failed')
  
  const reader = res.body?.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  let success = false
  
  while (reader) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    const lines = buffer.split('\n')
    buffer = lines.pop() || ''
    for (const line of lines) {
      if (line.startsWith('data: ')) {
        try {
          const data = JSON.parse(line.slice(6))
          if (data.type === 'output' && data.message) writeRaw(data.message)
          else if (data.type === 'done' && data.success) success = true
          else if (data.type === 'error') appendLog(data.message, 'error')
        } catch {}
      }
    }
  }
  return success
}

// 一键安装缺失依赖 - 分开处理 torch 和普通依赖
const installMissingDeps = async () => {
  if (!selectedPluginDeps.value?.missing?.length && !selectedPluginDeps.value?.mismatched?.length) return
  const allPkgs = [...(selectedPluginDeps.value.missing || []), ...(selectedPluginDeps.value.mismatched || []).map(m => m.split(' ')[0])]
  if (allPkgs.length === 0) return
  
  // 分离 torch 相关依赖和普通依赖
  const validPkgs = allPkgs.filter((p): p is string => !!p)
  const torchPkgs = validPkgs.filter(p => p.toLowerCase().startsWith('torch'))
  const otherPkgs = validPkgs.filter(p => !p.toLowerCase().startsWith('torch'))
  const torchReqs = selectedPlugin.value?.python?.torchRequirements
  
  isInstallingDeps.value = true
  showNewTaskDialog.value = false
  const notifyKey = notification.info(t('training.newTaskDialog.installingDeps'), { persistent: true })
  
  try {
    // 1. 先安装 torch（使用专用源）
    if (torchPkgs.length > 0 && torchReqs?.indexUrl) {
      appendLog(`Installing PyTorch with CUDA support...`, 'info')
      await runPipInstall(torchPkgs, torchReqs.indexUrl)
    } else if (torchPkgs.length > 0) {
      await runPipInstall(torchPkgs)
    }
    
    // 2. 安装其他依赖
    if (otherPkgs.length > 0) {
      await runPipInstall(otherPkgs)
    }
    
    appendLog(t('training.newTaskDialog.installDepsComplete'), 'info')
    notification.remove(notifyKey)
    notification.success(t('training.newTaskDialog.installDepsComplete'))
    await checkPluginDeps()
  } catch { 
    notification.remove(notifyKey)
    notification.error(t('training.newTaskDialog.installDepsFailed'))
    appendLog(t('training.newTaskDialog.installDepsFailed'), 'error')
  }
  isInstallingDeps.value = false
}

// 重新安装 CUDA 版本的 torch
const reinstallTorchCuda = async () => {
  const torchReqs = selectedPlugin.value?.python?.torchRequirements
  if (!torchReqs?.packages || !torchReqs?.indexUrl) {
    notification.error(t('training.newTaskDialog.noTorchConfig'))
    return
  }
  
  isInstallingDeps.value = true
  showNewTaskDialog.value = false
  const notifyKey = notification.info(t('training.newTaskDialog.reinstallingTorch'), { persistent: true })
  
  try {
    appendLog('Reinstalling PyTorch with CUDA support...', 'info')
    // 先卸载旧版本（只卸载插件声明的包，提取包名去掉版本号）
    const pkgsToUninstall = torchReqs.packages.map(p => p.split(/[<>=]/)[0]).filter(Boolean) as string[]
    for (const pkg of pkgsToUninstall) {
      await runPipUninstall([pkg])
    }
    // 安装 CUDA 版本
    await runPipInstall(torchReqs.packages, torchReqs.indexUrl)
    
    appendLog(t('training.newTaskDialog.installDepsComplete'), 'info')
    notification.remove(notifyKey)
    notification.success(t('training.newTaskDialog.torchCudaInstalled'))
    await checkPluginDeps()
  } catch {
    notification.remove(notifyKey)
    notification.error(t('training.newTaskDialog.installDepsFailed'))
  }
  isInstallingDeps.value = false
}

// 运行 pip uninstall
const runPipUninstall = async (pkgs: string[]) => {
  appendLog(`pip uninstall -y ${pkgs.join(' ')}`, 'cmd')
  const res = await fetch(`${API_BASE}/api/python/uninstall`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ package: pkgs.join(' ') })
  })
  if (!res.ok) throw new Error('Uninstall failed')
  // 读取并显示输出
  const reader = res.body?.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  while (reader) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    const lines = buffer.split('\n')
    buffer = lines.pop() || ''
    for (const line of lines) {
      if (line.startsWith('data: ')) {
        try {
          const data = JSON.parse(line.slice(6))
          if (data.type === 'output' && data.message) writeRaw(data.message)
        } catch {}
      }
    }
  }
}

// 检查模型下载状态
const checkModelStatus = async () => {
  if (!fullModelId.value) {
    modelDownloadStatus.value = null
    return
  }
  try {
    const res = await fetch(`${API_BASE}/api/training/model-status?modelId=${encodeURIComponent(fullModelId.value)}`)
    if (res.ok) modelDownloadStatus.value = await res.json()
  } catch { modelDownloadStatus.value = null }
}

// 打开模型目录
const openModelsDir = () => {
  const api = (window as any)?.electronAPI
  if (api?.openPath) api.openPath('models')
}

// 下载模型 - 使用通知组件显示进度
const downloadNotifyKey = ref<string | null>(null)

const downloadModel = async () => {
  if (!fullModelId.value) return
  isDownloadingModel.value = true
  
  // 创建一个持久通知，带取消按钮
  const modelName = fullModelId.value
  downloadNotifyKey.value = notification.info(
    t('training.newTaskDialog.downloadingModel', { model: modelName }) + ' 0%',
    { persistent: true, button: { text: t('training.newTaskDialog.cancelDownload'), onClick: cancelModelDownload } }
  )
  
  try {
    const downloadUrl = selectedModel.value?.isStandard ? '' : selectedModel.value?.downloadUrl
    const res = await fetch(`${API_BASE}/api/training/download-model`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ modelId: fullModelId.value, downloadUrl })
    })
    if (res.ok) {
      // 启动轮询检查下载进度
      startModelStatusPolling()
    }
  } catch { 
    notification.error(t('training.newTaskDialog.downloadModelFailed'))
    isDownloadingModel.value = false
  }
}

// 取消模型下载
const cancelModelDownload = async () => {
  if (fullModelId.value) {
    try {
      await fetch(`${API_BASE}/api/training/cancel-download`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ modelId: fullModelId.value })
      })
    } catch {}
  }
  stopModelStatusPolling()
  isDownloadingModel.value = false
  if (downloadNotifyKey.value) {
    notification.remove(downloadNotifyKey.value)
    downloadNotifyKey.value = null
  }
  notification.warning(t('training.newTaskDialog.downloadCancelled'))
}

// 轮询模型下载状态
const startModelStatusPolling = () => {
  stopModelStatusPolling()
  modelStatusPollTimer.value = window.setInterval(async () => {
    await checkModelStatus()
    const status = modelDownloadStatus.value
    if (status?.status === 'downloading' && downloadNotifyKey.value) {
      // 更新通知进度
      const percent = Math.round((status.progress || 0) * 100)
      notification.update(downloadNotifyKey.value, {
        message: t('training.newTaskDialog.downloadingModel', { model: fullModelId.value }) + ` ${percent}%`
      })
    } else if (status?.status === 'completed') {
      stopModelStatusPolling()
      isDownloadingModel.value = false
      if (downloadNotifyKey.value) {
        notification.remove(downloadNotifyKey.value)
        downloadNotifyKey.value = null
      }
      notification.success(t('training.newTaskDialog.modelDownloaded', { model: fullModelId.value }))
    } else if (status?.status === 'failed') {
      stopModelStatusPolling()
      isDownloadingModel.value = false
      if (downloadNotifyKey.value) {
        notification.remove(downloadNotifyKey.value)
        downloadNotifyKey.value = null
      }
      notification.error(t('training.newTaskDialog.downloadModelFailed'))
    }
  }, 1000)
}

const stopModelStatusPolling = () => {
  if (modelStatusPollTimer.value) {
    clearInterval(modelStatusPollTimer.value)
    modelStatusPollTimer.value = null
  }
}

// 是否可以开始训练
const canStartTraining = computed(() => {
  const f = newTaskForm.value
  if (!f.name.trim()) return false
  if (f.sourceType === 'trainset' && !f.trainsetId) return false
  if (f.sourceType === 'directory' && !f.datasetPath) return false
  if (!f.pluginId || !f.modelId) return false
  // 检查模型是否已下载
  if (modelDownloadStatus.value?.status !== 'completed') return false
  // 检查依赖是否满足
  if (selectedPluginDeps.value && (selectedPluginDeps.value.missing?.length > 0 || selectedPluginDeps.value.mismatched?.length > 0)) return false
  return true
})

// 监听对话框打开
watch(showNewTaskDialog, async (s) => {
  if (s) {
    newTaskForm.value.name = generateDefaultTaskName()
    await Promise.all([loadTrainingPlugins(), loadSavedTrainsets()])
    // 自动选择第一个插件
    const fp = filteredPlugins.value
    if (fp && fp.length > 0 && fp[0] && !newTaskForm.value.pluginId) {
      newTaskForm.value.pluginId = fp[0].id
    }
  } else {
    stopModelStatusPolling()
  }
})

// 监听训练类型变化
watch(() => newTaskForm.value.trainType, async () => {
  // 重置插件和模型选择
  newTaskForm.value.pluginId = ''
  newTaskForm.value.modelId = ''
  newTaskForm.value.modelVariant = 'n'
  selectedPluginDeps.value = null
  modelDownloadStatus.value = null
  // 自动选择第一个可用插件
  const fp = filteredPlugins.value
  if (fp && fp.length > 0 && fp[0]) {
    newTaskForm.value.pluginId = fp[0].id
  }
  // 重新检查依赖
  await checkPluginDeps()
})

// 监听插件变化
watch(() => newTaskForm.value.pluginId, async () => {
  newTaskForm.value.modelId = ''
  newTaskForm.value.modelVariant = 'n'
  modelDownloadStatus.value = null
  // 自动选择第一个可用模型
  const fm = filteredModels.value
  if (fm && fm.length > 0 && fm[0]) {
    newTaskForm.value.modelId = fm[0].id
  }
  // 重置插件参数为默认值
  const newParams: Record<string, any> = {}
  for (const param of pluginParamsSchema.value) {
    newParams[param.key] = param.default ?? (param.type === 'boolean' ? false : param.type === 'number' || param.type === 'integer' ? 0 : '')
  }
  pluginParams.value = newParams
  // 检查依赖
  await checkPluginDeps()
})

// 监听模型变化
watch([() => newTaskForm.value.modelId, () => newTaskForm.value.modelVariant], async () => {
  checkModelStatus()
  // 模型变化时也重新检查依赖
  await checkPluginDeps()
})

const browseDatasetPath = async () => {
  const api = (window as any)?.electronAPI
  if (api?.selectDirectory) {
    const p = await api.selectDirectory('')
    if (p) { newTaskForm.value.datasetPath = p; detectDatasetFormat() }
  }
}

const detectDatasetFormat = async () => {
  if (!newTaskForm.value.datasetPath) return
  isDetectingFormat.value = true
  detectedFormat.value = null
  try {
    const res = await fetch(`${API_BASE}/api/dataset/detect`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ rootPath: newTaskForm.value.datasetPath })
    })
    const data = await res.json()
    // 从检测结果中提取格式（取第一个支持的插件）
    if (data.results && data.results.length > 0) {
      detectedFormat.value = data.results[0].reason || data.results[0].pluginId
    }
  } catch { detectedFormat.value = null }
  isDetectingFormat.value = false
}

const startTraining = async () => {
  if (!canStartTraining.value) return
  
  const plugin = selectedPlugin.value
  if (!plugin) return
  
  // 检查必需字段
  if (!plugin.pluginPath) {
    notification.error(t('training.errors.pluginPathMissing'))
    console.error('Plugin path is missing:', plugin)
    return
  }
  
  isStartingTask.value = true
  
  try {
    // 生成唯一任务 ID
    const taskId = `task_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`
    
    // 构建训练参数
    const params: Record<string, any> = {
      // 基本参数
      trainType: newTaskForm.value.trainType,
      epochs: newTaskForm.value.epochs,
      batch: newTaskForm.value.batch,
      imgsz: newTaskForm.value.imgsz,
      trainRatio: newTaskForm.value.trainRatio / 100,
      // 数据集
      sourceType: newTaskForm.value.sourceType,
      trainsetId: newTaskForm.value.trainsetId,
      datasetPath: newTaskForm.value.datasetPath,
      // 模型
      modelId: newTaskForm.value.modelId,
      modelVariant: newTaskForm.value.modelVariant,
      modelPath: fullModelId.value ? `${fullModelId.value}.pt` : '',
      // 插件自定义参数
      ...pluginParams.value
    }
    
    const res = await fetch(`${API_BASE}/api/training/start`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        taskId,
        taskName: newTaskForm.value.name,
        pluginId: plugin.id,
        pluginPath: plugin.pluginPath,
        entry: plugin.entry || 'main.py',
        params
      })
    })
    
    if (res.ok) {
      notification.success(t('training.newTaskDialog.trainingStarted', { name: newTaskForm.value.name }))
      showNewTaskDialog.value = false
      appendLog(`Training task started: ${newTaskForm.value.name}`, 'info')
      await loadTrainingTasks()
    } else {
      const err = await res.json()
      notification.error(t('training.newTaskDialog.startFailed') + ': ' + (err.message || err.error))
    }
  } catch (e) {
    console.error('Start training failed:', e)
    notification.error(t('training.newTaskDialog.startFailed'))
  }
  
  isStartingTask.value = false
}

// ============ 环境管理 ============
const deployEnvironment = async () => {
  if (isDeploying.value) return
  isDeploying.value = true
  appendLog(`$ python -m venv ${pythonEnv.value.venvPath || '...'}`, 'cmd')

  // 关闭面板并提示开始部署
  showEnvDialog.value = false
  notification.info(t('training.envDialog.deployStarted'))

  try {
    const res = await fetch(`${API_BASE}/api/python/deploy`, { method: 'POST' })
    const data = await res.json()
    if (data.output) appendLog(data.output)
    if (data.success) {
      pythonEnv.value.deployed = true
      pythonEnv.value.venvPath = data.venvPath
      appendLog(`Virtual environment created at: ${data.venvPath}`)
    } else if (data.error) {
      appendLog(`Error: ${data.error}${data.details ? ' - ' + data.details : ''}`, 'error')
    }
  } catch (e) { appendLog(`Network error: ${e}`, 'error') }
  isDeploying.value = false
}
const installPackage = async () => {
  if (!packageToInstall.value.trim()) return
  const pkgName = packageToInstall.value.trim()
  isInstalling.value = true
  
  // 构建安装命令显示
  const mirrorArg = usePypiMirror.value && pypiMirrorUrl.value ? ` -i ${pypiMirrorUrl.value}` : ''
  appendLog(`$ pip install ${pkgName}${mirrorArg}`, 'cmd')
  
  // 使用fetch + ReadableStream来处理SSE (因为POST请求不能用EventSource)
  try {
    const reqBody: Record<string, unknown> = { package: pkgName }
    if (usePypiMirror.value && pypiMirrorUrl.value) {
      reqBody.mirror = pypiMirrorUrl.value
    }

    // 关闭面板并提示开始安装
    showEnvDialog.value = false
    notification.info(t('training.envDialog.installStarted'))

    const res = await fetch(`${API_BASE}/api/python/install`, { 
      method: 'POST', 
      headers: { 'Content-Type': 'application/json' }, 
      body: JSON.stringify(reqBody) 
    })
    
    if (!res.body) {
      appendLog('Error: No response body', 'error')
      isInstalling.value = false
      return
    }
    
    const reader = res.body.getReader()
    const decoder = new TextDecoder()
    let buffer = ''
    
    while (true) {
      const { done, value } = await reader.read()
      if (done) break
      
      buffer += decoder.decode(value, { stream: true })
      const lines = buffer.split('\n')
      buffer = lines.pop() || ''
      
      for (const line of lines) {
        if (line.startsWith('data: ')) {
          try {
            const data = JSON.parse(line.slice(6))
            if (data.type === 'output' && data.message) {
              // 原始输出，直接写入终端（保留进度条等）
              writeRaw(data.message)
            } else if (data.type === 'done') {
              terminal.value?.writeln('') // 换行
              if (data.success) {
                await loadInstalledPackages()
                packageToInstall.value = ''
              } else {
                appendLog(`Error: ${data.error || 'Installation failed'}`, 'error')
              }
            } else if (data.type === 'error') {
              appendLog(`Error: ${data.message}`, 'error')
            }
          } catch {}
        }
      }
    }
  } catch (e) { appendLog(`Network error: ${e}`, 'error') }
  isInstalling.value = false
}
const loadInstalledPackages = async () => {
  try {
    const res = await fetch(`${API_BASE}/api/python/packages`)
    const data = await res.json()
    installedPackages.value = data.packages || []
  } catch { installedPackages.value = [] }
}
const undeployEnvironment = async () => {
  isUndeploying.value = true
  const venvPath = pythonEnv.value.venvPath
  appendLog(`Removing virtual environment: ${venvPath}`)

  // 关闭面板和确认框，并提示开始卸载
  showUndeployConfirm.value = false
  showEnvDialog.value = false
  notification.info(t('training.envDialog.undeployStarted'))
  try {
    const res = await fetch(`${API_BASE}/api/python/undeploy`, { method: 'POST' })
    const data = await res.json()
    if (data.output) appendLog(data.output)
    if (data.success) {
      appendLog(`Virtual environment removed successfully`)
      pythonEnv.value.deployed = false
      pythonEnv.value.venvPath = ''
      installedPackages.value = []
      showUndeployConfirm.value = false
      showEnvDialog.value = false
    } else if (data.error) {
      appendLog(`Error: ${data.error}${data.details ? ' - ' + data.details : ''}`, 'error')
    }
  } catch (e) { appendLog(`Network error: ${e}`, 'error') }
  isUndeploying.value = false
}
const confirmUninstallPackage = (name: string) => { packageToUninstall.value = name }
const uninstallPackage = async () => {
  if (!packageToUninstall.value) return
  const pkgName = packageToUninstall.value
  isUninstalling.value = pkgName
  appendLog(`$ pip uninstall -y ${pkgName}`, 'cmd')
  try {
    const res = await fetch(`${API_BASE}/api/python/uninstall`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ package: pkgName }) })
    const data = await res.json()
    if (data.output) appendLog(data.output)
    if (data.success) {
      await loadInstalledPackages()
    } else if (data.error) {
      appendLog(`Error: ${data.error}${data.details ? '\n' + data.details : ''}`, 'error')
    }
  } catch (e) { appendLog(`Network error: ${e}`, 'error') }
  isUninstalling.value = null
  packageToUninstall.value = null
}

// ============ 初始化 ============
const loadPythonEnv = async () => {
  try {
    const res = await fetch(`${API_BASE}/api/python/status`)
    const data = await res.json()
    pythonEnv.value = { deployed: data.deployed || false, pythonVersion: data.pythonVersion || '', venvPath: data.venvPath || '' }
    if (data.deployed) loadInstalledPackages()
  } catch { /* ignore */ }
}
onMounted(async () => { 
  nextTick(() => initTerminal())
  loadPythonEnv()
  await loadTrainingPlugins() // 加载训练插件列表
  await checkTrainingPluginDeps() // 检查训练插件依赖状态
  loadTrainingHistory() // 加载历史训练记录
  startTasksRefresh() // WebSocket 连接，包含资源监控
})
onUnmounted(() => {
  stopTasksRefresh()
  terminal.value?.dispose()
})
</script>

<style scoped>
.training-view { display: flex; flex-direction: column; width: 100%; height: 100%; background: var(--color-bg-app); }
.top-bar { display: flex; height: 60px; border-bottom: 1px solid var(--color-border-subtle); background: var(--color-bg-sidebar); }
.resource-card { flex: 1; display: flex; align-items: center; justify-content: center; gap: 12px; padding: 10px 16px; border-right: 1px solid var(--color-border-subtle); }
.resource-card:last-child { border-right: none; }
.resource-icon { display: flex; align-items: center; justify-content: center; width: 36px; height: 36px; border-radius: 6px; background: var(--color-bg-app); color: var(--color-fg-muted); }
.resource-info { display: flex; flex-direction: column; align-items: flex-start; gap: 4px; min-width: 120px; }
.resource-text { font-size: 11px; color: var(--color-fg-muted); }
.resource-text.not-available { opacity: 0.5; }
.resource-status { display: flex; align-items: center; gap: 4px; }
.status-text { font-size: 12px; }
.status-text.deployed { color: #52c41a; }
.status-text.not-deployed { color: #faad14; }
.help-icon { color: var(--color-fg-muted); cursor: help; display: flex; opacity: 0.6; }
.progress-bar { width: 100%; height: 6px; background: var(--color-border-subtle); border-radius: 3px; overflow: hidden; }
.progress-bar.small { height: 4px; border-radius: 2px; }
.progress-fill { height: 100%; border-radius: inherit; transition: width 0.3s; }
.progress-fill.success { background: #52c41a; }
.progress-fill.warning { background: #faad14; }
.progress-fill.error { background: #ff4d4f; }
.main-content { flex: 1; display: flex; overflow: hidden; }
.left-panel { flex: 1; display: flex; flex-direction: column; overflow: hidden; border-right: 1px solid var(--color-border-subtle); }
.tasks-section { display: flex; flex-direction: column; overflow: hidden; }
.section-header { display: flex; align-items: center; justify-content: space-between; height: 36px; padding: 0 10px; border-bottom: 1px solid var(--color-border-subtle); background: var(--color-bg-sidebar); }
.section-title { font-size: 12px; font-weight: 500; color: var(--color-fg); }
.tasks-body { flex: 1; overflow-y: auto; padding: 10px; background: var(--color-bg-app); position: relative; }
.task-list { display: flex; flex-wrap: wrap; gap: 8px; align-content: flex-start; }
.task-card {
  padding: 10px;
  background: var(--color-bg-sidebar);
  border: 1px solid var(--color-border-subtle);
  border-radius: 6px;
  flex: 0 1 calc(33.33% - 8px);
  min-width: 260px;
  max-width: 360px;
  box-sizing: border-box;
}
.task-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 6px; }
.task-name-row { display: flex; align-items: center; gap: 6px; }
.task-name { font-size: 13px; font-weight: 500; color: var(--color-fg); }
.task-badge { display: inline-flex; align-items: center; gap: 3px; padding: 1px 6px; border-radius: 8px; font-size: 10px; }
.badge-running { background: rgba(24, 144, 255, 0.15); color: #1890ff; }
.badge-pending { background: rgba(250, 173, 20, 0.15); color: #faad14; }
.badge-error { background: rgba(255, 77, 79, 0.15); color: #ff4d4f; }
.badge-completed { background: rgba(82, 196, 26, 0.15); color: #52c41a; }
.badge-failed { background: rgba(255, 77, 79, 0.15); color: #ff4d4f; }
.badge-success { background: rgba(82, 196, 26, 0.15); color: #52c41a; }
.task-menu-wrapper { position: relative; }
.task-menu { position: absolute; right: 0; top: 100%; background: var(--color-bg-header); border: 1px solid var(--color-border-subtle); border-radius: 4px; box-shadow: 0 4px 12px rgba(0,0,0,0.25); z-index: 10; min-width: 90px; }
.task-menu button { display: block; width: 100%; padding: 6px 10px; text-align: left; background: none; border: none; cursor: pointer; font-size: 12px; color: var(--color-fg); }
.task-menu button:hover { background: var(--color-bg-sidebar-hover); }
.task-progress-row { display: flex; align-items: center; justify-content: space-between; margin-bottom: 4px; font-size: 11px; color: var(--color-fg-muted); }
.epoch-text { font-family: monospace; }
.progress-percent { font-weight: 500; }
.task-metrics { display: grid; grid-template-columns: repeat(3, 1fr); gap: 4px; margin-top: 8px; padding-top: 8px; border-top: 1px solid var(--color-border-subtle); }
.metrics-group { display: contents; }
.metric-item { display: flex; flex-direction: column; align-items: center; padding: 4px 2px; background: var(--color-bg-app); border-radius: 4px; }
.metric-value { font-family: monospace; font-size: 11px; font-weight: 600; color: var(--color-fg); line-height: 1.2; order: 1; }
.metric-label { font-size: 8px; color: var(--color-fg-muted); margin-bottom: 2px; text-transform: uppercase; order: 0; }
.resize-handle { height: 4px; background: var(--color-border-subtle); cursor: ns-resize; transition: background 0.2s; }
.resize-handle:hover { background: var(--color-accent); }
.logs-section { display: flex; flex-direction: column; min-height: 100px; background: var(--color-bg-sidebar); }
.log-actions { display: flex; align-items: center; gap: 8px; }
.terminal-container { flex: 1; overflow: hidden; background: #0d0d0d; }
[data-theme='light'] .terminal-container { background: #f5f5f5; }
/* xterm 自定义滚动条 (类似 VSCode) */
.terminal-container :deep(.xterm-viewport) {
  scrollbar-width: thin;
  scrollbar-color: rgba(121, 121, 121, 0.4) transparent;
}
.terminal-container :deep(.xterm-viewport::-webkit-scrollbar) {
  width: 10px;
}
.terminal-container :deep(.xterm-viewport::-webkit-scrollbar-track) {
  background: transparent;
}
.terminal-container :deep(.xterm-viewport::-webkit-scrollbar-thumb) {
  background: rgba(121, 121, 121, 0.4);
  border-radius: 5px;
  border: 2px solid transparent;
  background-clip: padding-box;
}
.terminal-container :deep(.xterm-viewport::-webkit-scrollbar-thumb:hover) {
  background: rgba(121, 121, 121, 0.7);
  border: 2px solid transparent;
  background-clip: padding-box;
}
.right-panel { width: 220px; display: flex; flex-direction: column; background: var(--color-bg-sidebar); }
.history-body { flex: 1; overflow-y: auto; padding: 10px; position: relative; }
.history-list { display: flex; flex-direction: column; gap: 6px; }
.history-item { padding: 8px; background: var(--color-bg-app); border: 1px solid var(--color-border-subtle); border-radius: 4px; }
.history-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 3px; }
.history-name { font-size: 12px; font-weight: 500; color: var(--color-fg); flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.history-menu-wrapper { position: relative; flex-shrink: 0; }
.history-menu { position: absolute; right: 0; top: 100%; background: var(--color-bg-header); border: 1px solid var(--color-border-subtle); border-radius: 4px; box-shadow: 0 4px 12px rgba(0,0,0,0.25); z-index: 10; min-width: 100px; }
.history-menu button { display: flex; align-items: center; gap: 6px; width: 100%; padding: 6px 10px; text-align: left; background: none; border: none; cursor: pointer; font-size: 12px; color: var(--color-fg); }
.history-menu button:hover { background: var(--color-bg-sidebar-hover); }
.history-meta { font-size: 10px; color: var(--color-fg-muted); }
.empty-state {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 100%;
  padding: 30px 16px;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
}
.empty-icon { color: var(--color-fg-muted); opacity: 0.4; margin-bottom: 10px; }
.empty-text { font-size: 13px; color: var(--color-fg-muted); margin: 0 0 4px; }
.empty-hint { font-size: 11px; color: var(--color-fg-muted); opacity: 0.6; margin: 0; }
/* Buttons */
.btn { padding: 5px 10px; border: 1px solid var(--color-border-subtle); border-radius: 4px; background: var(--color-bg-header); color: var(--color-fg); cursor: pointer; font-size: 12px; display: inline-flex; align-items: center; gap: 4px; transition: all 0.15s; }
.btn:hover { background: var(--color-bg-sidebar-hover); border-color: var(--color-fg-muted); }
.btn:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-primary { background: var(--color-accent); border-color: var(--color-accent); color: #fff; }
.btn-primary:hover { background: var(--color-accent-hover); border-color: var(--color-accent-hover); }
.btn-danger { background: #dc2626; border-color: #dc2626; color: #fff; }
.btn-danger:hover { background: #b91c1c; border-color: #b91c1c; }
.btn-small { padding: 3px 7px; font-size: 11px; }
.btn-icon { background: none; border: none; padding: 4px; cursor: pointer; color: var(--color-fg-muted); display: flex; align-items: center; border-radius: 3px; }
.btn-icon:hover { color: var(--color-fg); background: var(--color-bg-sidebar-hover); }
.btn-text { background: none; border: none; padding: 0; cursor: pointer; color: var(--color-accent); font-size: 11px; display: flex; align-items: center; gap: 3px; }
.btn-text:hover { text-decoration: underline; }
/* Modal */
.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.6); display: flex; align-items: center; justify-content: center; z-index: 1000; }
.modal { background: var(--color-bg-header); border: 1px solid var(--color-border-subtle); border-radius: 8px; padding: 20px; max-width: 90vw; max-height: 90vh; overflow: auto; box-shadow: 0 12px 40px rgba(0,0,0,0.4); }
.modal-title { margin: 0 0 14px; font-size: 16px; font-weight: 500; color: var(--color-fg); }
.modal-footer { display: flex; justify-content: flex-end; gap: 10px; margin-top: 20px; }
.env-modal { width: 600px; }
.new-task-modal { width: 520px; }
.new-task-modal-v2 { width: 580px; max-height: 85vh; display: flex; flex-direction: column; }
.new-task-modal-v2 .new-task-form { flex: 1; overflow-y: auto; max-height: 60vh; padding-right: 8px; }
.split-row { display: flex; align-items: center; gap: 8px; }
.split-slider { flex: 1; height: 4px; cursor: pointer; }
.split-label { font-size: 11px; color: var(--color-fg-muted); min-width: 30px; }
.split-value { font-size: 11px; font-weight: 500; color: var(--color-fg); min-width: 35px; }
.plugin-select-row { display: flex; align-items: center; gap: 8px; }
.plugin-select-row .select { flex: 1; }
.deps-status { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; font-size: 12px; }
.deps-status .text-success { color: #52c41a; }
.deps-status .text-error { color: #ff4d4f; }
.deps-status .text-warning { color: #faad14; }
.model-select-row { display: flex; align-items: center; gap: 8px; }
.model-selects { display: flex; align-items: center; gap: 6px; flex: 1; min-width: 0; }
/* 4:4:2 布局：模型 4 份，版本 4 份，右侧操作区 2 份 */
.select-model { flex: 4 1 0; min-width: 0; }
.select-variant { flex: 4 1 0; min-width: 0; }
.model-actions { display: flex; align-items: center; gap: 4px; flex: 2 0 0; justify-content: flex-end; min-width: 0; }
.model-status { display: flex; align-items: center; gap: 4px; font-size: 11px; }
.model-status--ok { color: #52c41a; }
.model-status--downloading { color: var(--color-accent); }
.btn-download { white-space: nowrap; }
.btn-browse { white-space: nowrap; min-width: fit-content; }
.form-hint { font-size: 11px; margin-top: 4px; }
.confirm-modal { width: 400px; }
.confirm-text { margin: 0 0 8px; font-size: 13px; color: var(--color-fg-muted); line-height: 1.5; }
.env-dialog-content { display: flex; gap: 16px; }
.env-left { flex: 1.5; display: flex; flex-direction: column; gap: 12px; min-width: 220px; }
.env-right { flex: 1; display: flex; flex-direction: column; min-width: 0; }
.env-row-actions { margin-top: auto; padding-top: 8px; }
.env-row { display: flex; flex-direction: column; gap: 5px; }
.env-label { font-size: 12px; font-weight: 500; color: var(--color-fg); }
.env-value { display: flex; align-items: center; font-size: 12px; color: var(--color-fg-muted); flex-wrap: wrap; gap: 6px; }
.venv-path { font-family: monospace; font-size: 11px; word-break: break-all; }
.text-warning { color: #faad14; }
.text-muted { color: var(--color-fg-muted); opacity: 0.6; }
.warning-tag { background: rgba(250, 173, 20, 0.15); color: #faad14; padding: 2px 6px; border-radius: 3px; font-size: 10px; }
.install-input-row { display: flex; gap: 6px; }
.btn-env-action { padding: 6px 12px; font-size: 12px; }
.btn-install { white-space: nowrap; min-width: 70px; padding: 6px 14px; }
.mirror-row { display: flex; flex-direction: column; gap: 6px; }
.checkbox-inline { display: flex; align-items: center; gap: 6px; font-size: 12px; color: var(--color-fg); cursor: pointer; }
.checkbox-inline input { margin: 0; }
.mirror-input { font-size: 11px; }
.packages-list { flex: 1; max-height: 280px; overflow-y: auto; border: 1px solid var(--color-border-subtle); border-radius: 4px; margin-top: 6px; background: var(--color-bg-app); }
.package-items { padding: 6px; }
.package-item { display: flex; align-items: center; justify-content: space-between; padding: 4px 6px; font-size: 11px; border-radius: 3px; gap: 8px; }
.package-item:hover { background: var(--color-bg-sidebar-hover); }
.package-info { flex: 1; min-width: 0; display: flex; align-items: center; gap: 6px; }
.package-name { color: var(--color-fg); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.package-version { color: var(--color-fg-muted); font-family: monospace; flex-shrink: 0; }
.btn-uninstall { flex-shrink: 0; opacity: 0; transition: opacity 0.15s; }
.package-item:hover .btn-uninstall { opacity: 1; }
.btn-uninstall:hover { color: #dc2626 !important; }
.packages-empty { display: flex; align-items: center; justify-content: center; height: 80px; color: var(--color-fg-muted); font-size: 12px; }
/* Form */
.new-task-form { display: flex; flex-direction: column; gap: 14px; }
.form-row { display: flex; flex-direction: column; gap: 5px; }
.form-label { font-size: 12px; font-weight: 500; color: var(--color-fg); }
.input, .select { padding: 7px 10px; border: 1px solid var(--color-border-subtle); border-radius: 4px; background: var(--color-bg-app); color: var(--color-fg); font-size: 12px; width: 100%; box-sizing: border-box; }
.input:focus, .select:focus { outline: none; border-color: var(--color-accent); }
.source-cards { display: flex; gap: 10px; }
.source-card { flex: 1; display: flex; flex-direction: column; align-items: center; gap: 6px; padding: 12px; border: 2px solid var(--color-border-subtle); border-radius: 6px; cursor: pointer; transition: all 0.2s; color: var(--color-fg-muted); }
.source-card:hover { border-color: var(--color-accent); }
.source-card.active { border-color: var(--color-accent); background: rgba(59, 130, 246, 0.1); color: var(--color-accent); }
.source-card span { font-size: 12px; }
.path-input-row { display: flex; gap: 6px; }
.detected-format { display: flex; align-items: center; gap: 6px; }
.format-tag { background: rgba(82, 196, 26, 0.15); color: #52c41a; padding: 2px 6px; border-radius: 3px; font-size: 11px; font-weight: 500; }
.radio-group { display: flex; gap: 14px; }
.radio-label { display: flex; align-items: center; gap: 4px; font-size: 12px; color: var(--color-fg); cursor: pointer; }
.hyperparams-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 10px; }
.param-item { display: flex; flex-direction: column; gap: 3px; }
.param-item label { font-size: 11px; color: var(--color-fg-muted); }
/* Switch 开关样式 */
.switch { position: relative; display: inline-block; width: 36px; height: 20px; cursor: pointer; }
.switch input { opacity: 0; width: 0; height: 0; }
.switch .slider { position: absolute; inset: 0; background: var(--color-bg-secondary); border-radius: 10px; transition: 0.2s; }
.switch .slider::before { content: ''; position: absolute; width: 14px; height: 14px; left: 3px; bottom: 3px; background: #fff; border-radius: 50%; transition: 0.2s; }
.switch input:checked + .slider { background: var(--color-accent); }
.switch input:checked + .slider::before { transform: translateX(16px); }
/* Header Actions */
.header-actions { display: flex; gap: 6px; }
/* Trainset Modal */
.trainset-modal { width: 1000px; height: 580px; max-width: 95vw; max-height: 90vh; display: flex; flex-direction: column; }
.trainset-content { display: flex; gap: 16px; flex: 1; min-height: 0; }
.trainset-tree { flex: 1.2; display: flex; flex-direction: column; min-width: 320px; border: 1px solid var(--color-border-subtle); border-radius: 6px; background: var(--color-bg-app); }
.trainset-tree-header { padding: 10px 12px; font-size: 12px; font-weight: 500; color: var(--color-fg); border-bottom: 1px solid var(--color-border-subtle); flex-shrink: 0; }
.trainset-tree-body { flex: 1; overflow-y: auto; padding: 8px; min-height: 0; }
.tree-loading, .tree-empty { display: flex; align-items: center; justify-content: center; height: 100px; color: var(--color-fg-muted); font-size: 12px; gap: 8px; }
.ts-tree-node { margin-bottom: 2px; }
.ts-tree-row { display: flex; align-items: center; gap: 6px; padding: 4px 6px; border-radius: 4px; cursor: pointer; font-size: 12px; color: var(--color-fg); flex-wrap: wrap; }
.ts-tree-row:hover { background: var(--color-bg-sidebar-hover); }
.ts-tree-row--project { font-weight: 500; }
.ts-tree-row--version { padding-left: 20px; }
.ts-tree-row--category { padding-left: 40px; }
.ts-tree-row--category input { margin: 0; cursor: pointer; flex-shrink: 0; }
.ts-tree-meta { font-size: 10px; color: var(--color-fg-muted); margin-left: auto; }
.ts-tree-children { margin-left: 8px; }
.cat-name { flex-shrink: 0; }
.cat-type-tag { font-size: 10px; padding: 1px 5px; border-radius: 3px; background: rgba(100, 100, 100, 0.2); color: var(--color-fg-muted); flex-shrink: 0; }
.cat-stats { font-size: 10px; color: var(--color-fg-muted); opacity: 0.8; }
.ts-tree-row--disabled { opacity: 0.4; cursor: not-allowed; }
.ts-tree-row--disabled input { cursor: not-allowed; }
.trainset-selected { padding: 8px 12px; border-top: 1px solid var(--color-border-subtle); font-size: 11px; color: var(--color-fg-muted); background: var(--color-bg-sidebar); flex-shrink: 0; display: flex; justify-content: space-between; align-items: center; }
.trainset-stats-detail { font-size: 10px; color: var(--color-accent); }
.trainset-editor { flex: 1; display: flex; flex-direction: column; min-width: 200px; border: 1px solid var(--color-border-subtle); border-radius: 6px; background: var(--color-bg-app); }
.trainset-editor-header { padding: 10px 12px; font-size: 12px; font-weight: 500; color: var(--color-fg); border-bottom: 1px solid var(--color-border-subtle); }
.trainset-editor-body { flex: 1; padding: 12px; display: flex; flex-direction: column; gap: 12px; }
.trainset-editor-actions { padding: 12px; border-top: 1px solid var(--color-border-subtle); display: flex; gap: 8px; }
.categories-preview { border: 1px solid var(--color-border-subtle); border-radius: 4px; background: var(--color-bg-sidebar); min-height: 80px; max-height: 150px; overflow-y: auto; }
.preview-empty { display: flex; align-items: center; justify-content: center; height: 80px; color: var(--color-fg-muted); font-size: 12px; }
.preview-tags { padding: 8px; display: flex; flex-wrap: wrap; gap: 6px; }
.preview-tag { display: inline-flex; align-items: center; gap: 4px; padding: 3px 8px; background: var(--color-accent); color: #fff; border-radius: 12px; font-size: 11px; flex-wrap: wrap; }
.tag-source { font-size: 9px; opacity: 0.7; margin-left: 2px; }
.tag-remove { background: none; border: none; padding: 0; cursor: pointer; color: rgba(255,255,255,0.7); display: flex; }
.tag-remove:hover { color: #fff; }
.trainset-list { width: 200px; display: flex; flex-direction: column; border: 1px solid var(--color-border-subtle); border-radius: 6px; background: var(--color-bg-app); }
.trainset-list-header { padding: 10px 12px; font-size: 12px; font-weight: 500; color: var(--color-fg); border-bottom: 1px solid var(--color-border-subtle); }
.trainset-list-body { flex: 1; overflow-y: auto; max-height: 360px; }
.list-empty { display: flex; align-items: center; justify-content: center; height: 100px; color: var(--color-fg-muted); font-size: 12px; }
.trainset-items { padding: 6px; }
.trainset-item { display: flex; align-items: center; justify-content: space-between; padding: 6px 8px; border-radius: 4px; margin-bottom: 4px; cursor: pointer; }
.trainset-item:hover { background: var(--color-bg-sidebar-hover); }
.trainset-item--active { background: rgba(59, 130, 246, 0.15); border: 1px solid var(--color-accent); }
.trainset-item-info { flex: 1; min-width: 0; }
.trainset-item-name { display: block; font-size: 12px; color: var(--color-fg); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.trainset-item-meta { font-size: 10px; color: var(--color-fg-muted); }
.trainset-item-actions { display: flex; gap: 2px; opacity: 0; transition: opacity 0.15s; }
.trainset-item:hover .trainset-item-actions { opacity: 1; }
/* Animations */
@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
.spin { animation: spin 1s linear infinite; }
/* Custom Scrollbar */
.tasks-body, .log-container, .history-body, .packages-list, .modal {
  scrollbar-width: thin;
  scrollbar-color: rgba(255,255,255,0.15) transparent;
}
.tasks-body::-webkit-scrollbar, .log-container::-webkit-scrollbar, .history-body::-webkit-scrollbar, .packages-list::-webkit-scrollbar, .modal::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}
.tasks-body::-webkit-scrollbar-track, .log-container::-webkit-scrollbar-track, .history-body::-webkit-scrollbar-track, .packages-list::-webkit-scrollbar-track, .modal::-webkit-scrollbar-track {
  background: transparent;
}
.tasks-body::-webkit-scrollbar-thumb, .log-container::-webkit-scrollbar-thumb, .history-body::-webkit-scrollbar-thumb, .packages-list::-webkit-scrollbar-thumb, .modal::-webkit-scrollbar-thumb {
  background: rgba(255,255,255,0.15);
  border-radius: 3px;
}
.tasks-body::-webkit-scrollbar-thumb:hover, .log-container::-webkit-scrollbar-thumb:hover, .history-body::-webkit-scrollbar-thumb:hover, .packages-list::-webkit-scrollbar-thumb:hover, .modal::-webkit-scrollbar-thumb:hover {
  background: rgba(255,255,255,0.25);
}
</style>
