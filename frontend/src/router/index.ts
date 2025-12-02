import { createRouter, createWebHashHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
import App from '../App.vue'
import UiPage from '../views/UiPage.vue'
import HomePage from '../views/HomePage.vue'
import ProjectView from '../views/ProjectView.vue'
import DatasetView from '../views/DatasetView.vue'
import TrainingView from '../views/TrainingView.vue'
import PluginsView from '../views/PluginsView.vue'
import PythonEnvView from '../views/PythonEnvView.vue'
import HelpView from '../views/HelpView.vue'
import ModelInferenceWindow from '../views/ModelInferenceWindow.vue'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    component: App,
    children: [
      {
        path: '',
        name: 'Home',
        component: HomePage
      },
      {
        path: 'ui',
        name: 'UI',
        component: UiPage
      },
      {
        path: 'project',
        name: 'Project',
        component: ProjectView
      },
      {
        path: 'dataset',
        name: 'Dataset',
        component: DatasetView
      },
      {
        path: 'training',
        name: 'Training',
        component: TrainingView
      },
      {
        path: 'plugins',
        name: 'Plugins',
        component: PluginsView
      },
      {
        path: 'python-env',
        name: 'PythonEnv',
        component: PythonEnvView
      },
      {
        path: 'help',
        name: 'Help',
        component: HelpView
      }
    ]
  },
  // 模型推理小窗口（独立路由）
  {
    path: '/inference',
    name: 'ModelInference',
    component: ModelInferenceWindow
  }
]

const router = createRouter({
  history: createWebHashHistory(),
  routes
})

export default router

