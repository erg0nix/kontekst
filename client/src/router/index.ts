import { createRouter, createWebHistory } from 'vue-router'
import DashboardView from '../views/DashboardView.vue'
import AgentsView from '../views/AgentsView.vue'
import AgentDetailView from '../views/AgentDetailView.vue'
import SessionsView from '../views/SessionsView.vue'
import SessionDetailView from '../views/SessionDetailView.vue'
import SkillsView from '../views/SkillsView.vue'
import SkillDetailView from '../views/SkillDetailView.vue'

const routes = [
  {
    path: '/',
    name: 'dashboard',
    component: DashboardView,
  },
  {
    path: '/agents',
    name: 'agents',
    component: AgentsView,
  },
  {
    path: '/agents/:name',
    name: 'agent-detail',
    component: AgentDetailView,
  },
  {
    path: '/sessions',
    name: 'sessions',
    component: SessionsView,
  },
  {
    path: '/sessions/:id',
    name: 'session-detail',
    component: SessionDetailView,
  },
  {
    path: '/skills',
    name: 'skills',
    component: SkillsView,
  },
  {
    path: '/skills/:name',
    name: 'skill-detail',
    component: SkillDetailView,
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

export default router
