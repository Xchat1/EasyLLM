<template>
  <div class="dashboard-page flex flex-col gap-6 p-3.5 sm:p-5 max-w-[1700px] mx-auto">
    <!-- 1. Hero Section -->
    <section class="dashboard-hero">
      <div class="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
        <div class="max-w-3xl space-y-3">
          <div class="dashboard-hero__badge">
            <span>✨</span>
            <span>多渠道聚合总览与额度看板</span>
          </div>
          <div>
            <h1 class="dashboard-hero__title">总览与额度看板</h1>
            <p class="dashboard-hero__desc">
              集中监控 Codex (OpenAI)、Google Antigravity 与 Cursor 三大渠道的所有账号配额水位、健康度、账单重置周期与实时调用。
            </p>
          </div>
        </div>

        <div class="stable-action-row lg:justify-end">
          <button
            class="btn flex items-center gap-1.5 transition-colors"
            :class="privacyMode ? 'bg-indigo-500/20 text-indigo-300 border-indigo-500/40 shadow-sm' : 'btn-secondary'"
            @click="togglePrivacyMode"
            :title="privacyMode ? '已开启隐私模式：点击显示账号邮箱与敏感信息' : '一键开启隐私模式：隐藏账号ID、邮箱、名称与头像'"
          >
            <svg v-if="privacyMode" class="w-4 h-4 text-indigo-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21"/>
            </svg>
            <svg v-else class="w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"/>
            </svg>
            <span>{{ privacyMode ? '隐私已开启' : '一键隐私模式' }}</span>
          </button>
          <button class="btn btn-secondary flex items-center gap-1.5" :disabled="loading" @click="loadDashboard">
            <svg class="w-4 h-4" :class="{ 'animate-spin': loading }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
            </svg>
            <span>{{ loading ? '刷新中...' : '刷新总览' }}</span>
          </button>
          <div class="flex items-center gap-1.5 pl-2 border-l border-gray-800">
            <button class="btn btn-secondary btn-sm" @click="router.push('/codex')">🤖 Codex</button>
            <button class="btn btn-secondary btn-sm" @click="router.push('/antigravity')">🚀 Antigravity</button>
            <button class="btn btn-secondary btn-sm" @click="router.push('/cursor')">⚡ Cursor</button>
          </div>
        </div>
      </div>
    </section>

    <!-- 2. Multi-channel KPI Cards -->
    <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
      <!-- Card 1: Codex -->
      <div
        class="card dashboard-stat-card p-4 select-none cursor-pointer hover:border-blue-500/40 transition-colors"
        @click="selectedChannel = 'codex'"
      >
        <div class="flex items-center justify-between">
          <div class="dashboard-stat-card__label flex items-center gap-1.5 text-blue-400">
            <span>🤖</span>
            <span>Codex / OpenAI</span>
          </div>
          <span class="text-[11px] font-mono px-1.5 py-0.5 rounded bg-blue-500/10 text-blue-300 border border-blue-500/20">
            {{ pool.codex_api_service ? '代理服务运行' : '未启用' }}
          </span>
        </div>
        <div class="dashboard-stat-card__value text-blue-100">
          {{ codexAccounts.length }} <span class="text-sm font-normal text-gray-400">个账号</span>
        </div>
        <div class="dashboard-stat-card__sub text-xs text-gray-400 flex items-center justify-between">
          <span>OAuth: {{ oauthAccounts.length }} (活跃 {{ activeOAuthCount }})</span>
          <span>API: {{ apiAccounts.length }}</span>
        </div>
      </div>

      <!-- Card 2: Antigravity -->
      <div
        class="card dashboard-stat-card p-4 select-none cursor-pointer hover:border-amber-500/40 transition-colors"
        @click="selectedChannel = 'antigravity'"
      >
        <div class="flex items-center justify-between">
          <div class="dashboard-stat-card__label flex items-center gap-1.5 text-amber-400">
            <span>🚀</span>
            <span>Google Antigravity</span>
          </div>
          <span
            v-if="activeAntigravityAccount"
            class="text-[11px] font-mono px-1.5 py-0.5 rounded bg-amber-500/10 text-amber-300 border border-amber-500/20 truncate max-w-[130px]"
            :title="privacyMode ? '已隐藏' : activeAntigravityAccount.email"
          >
            {{ privacyMode ? '🔒 已激活 (脱敏)' : (activeAntigravityAccount.display_name || activeAntigravityAccount.email) }}
          </span>
          <span v-else class="text-[11px] text-gray-500">无激活账号</span>
        </div>
        <div class="dashboard-stat-card__value text-amber-100">
          {{ antigravityAccounts.length }} <span class="text-sm font-normal text-gray-400">个账号</span>
        </div>
        <div class="dashboard-stat-card__sub text-xs text-gray-400 flex items-center justify-between">
          <span>Claude 5h 均值: <b class="text-amber-300 font-mono">{{ avgAntigravityClaude != null ? `${avgAntigravityClaude}%` : '--' }}</b></span>
          <span>Gemini 5h: <b class="text-amber-300 font-mono">{{ avgAntigravityGemini != null ? `${avgAntigravityGemini}%` : '--' }}</b></span>
        </div>
      </div>

      <!-- Card 3: Cursor -->
      <div
        class="card dashboard-stat-card p-4 select-none cursor-pointer hover:border-purple-500/40 transition-colors"
        @click="selectedChannel = 'cursor'"
      >
        <div class="flex items-center justify-between">
          <div class="dashboard-stat-card__label flex items-center gap-1.5 text-purple-400">
            <span>⚡</span>
            <span>Cursor</span>
          </div>
          <span
            v-if="activeCursorAccount"
            class="text-[11px] font-mono px-1.5 py-0.5 rounded bg-purple-500/10 text-purple-300 border border-purple-500/20 truncate max-w-[130px]"
            :title="privacyMode ? '已隐藏' : activeCursorAccount.email"
          >
            {{ privacyMode ? '🔒 已激活 (脱敏)' : (activeCursorAccount.display_name || activeCursorAccount.email) }}
          </span>
          <span v-else class="text-[11px] text-gray-500">无激活账号</span>
        </div>
        <div class="dashboard-stat-card__value text-purple-100">
          {{ cursorAccounts.length }} <span class="text-sm font-normal text-gray-400">个账号</span>
        </div>
        <div class="dashboard-stat-card__sub text-xs text-gray-400 flex items-center justify-between">
          <span>Fast 剩余: <b class="text-purple-300 font-mono">{{ totalCursorFastRemaining }} 次</b></span>
          <span>超额按量: <b class="text-indigo-300 font-mono">${{ totalCursorOnDemandSpend }}</b></span>
        </div>
      </div>

      <!-- Card 4: Relay & Calls -->
      <div
        class="card dashboard-stat-card p-4 select-none cursor-pointer hover:border-emerald-500/40 transition-colors"
        @click="router.push('/relay')"
      >
        <div class="flex items-center justify-between">
          <div class="dashboard-stat-card__label flex items-center gap-1.5 text-emerald-400">
            <span>🔀</span>
            <span>Relay 转发监控</span>
          </div>
          <span class="text-[11px] font-mono px-1.5 py-0.5 rounded bg-emerald-500/10 text-emerald-300 border border-emerald-500/20">
            {{ relayUsage.upstream_configured ? '上游已配' : '未配置上游' }}
          </span>
        </div>
        <div class="dashboard-stat-card__value text-emerald-100">
          {{ formatTokens(relayUsage.usage?.total_tokens) }} <span class="text-sm font-normal text-gray-400">Tokens</span>
        </div>
        <div class="dashboard-stat-card__sub text-xs text-gray-400 flex items-center justify-between">
          <span>请求: {{ relayUsage.usage?.request_count || 0 }} 次</span>
          <span>最近: <span class="font-mono text-emerald-300">{{ relayUsage.usage?.last_model || '暂无' }}</span></span>
        </div>
      </div>
    </div>

    <!-- 3. 【核心】所有账号额度看板 (Unified All Accounts Quota Board) -->
    <section class="card p-5 space-y-4 shadow-lg border-gray-800/90">
      <!-- Board Header -->
      <div class="flex flex-col lg:flex-row lg:items-center lg:justify-between gap-4 border-b border-gray-800/80 pb-4">
        <div>
          <div class="flex items-center gap-2.5">
            <span class="text-2xl">📊</span>
            <div>
              <h2 class="text-lg font-bold text-white flex items-center gap-2">
                <span>所有账号额度看板</span>
                <span class="px-2 py-0.5 text-xs font-semibold rounded-full bg-blue-500/20 text-blue-300 border border-blue-500/30">
                  共 {{ unifiedAccounts.length }} 个账号
                </span>
              </h2>
              <p class="text-xs text-gray-400 mt-0.5">
                汇总三大渠道的全部账号，直观对比配额剩余、滑动窗口重置时间、账单周期与运行状态。
              </p>
            </div>
          </div>
        </div>

        <div class="flex flex-wrap items-center gap-2.5">
          <!-- 一键隐私模式 -->
          <button
            @click="togglePrivacyMode"
            class="btn btn-sm flex items-center gap-1.5 transition-colors shrink-0"
            :class="privacyMode ? 'bg-indigo-500/20 text-indigo-300 border-indigo-500/40 shadow-sm' : 'btn-secondary'"
            :title="privacyMode ? '隐私模式已开启：点击显示完整账号与敏感信息' : '一键开启隐私模式：全局脱敏隐藏账号ID、邮箱与名称'"
          >
            <svg v-if="privacyMode" class="w-3.5 h-3.5 text-indigo-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21"/>
            </svg>
            <svg v-else class="w-3.5 h-3.5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"/>
            </svg>
            <span>{{ privacyMode ? '隐私已开启' : '一键隐私模式' }}</span>
          </button>

          <!-- 视图模式切换 -->
          <div class="flex items-center rounded-xl border border-blue-500/30 bg-blue-500/10 p-0.5 shrink-0">
            <button
              @click="setViewMode('table')"
              class="px-2.5 py-1 text-xs font-medium rounded-lg transition-colors flex items-center gap-1.5"
              :class="viewMode === 'table' ? 'bg-blue-600 text-white font-bold shadow-sm' : 'text-blue-200/80 hover:text-white'"
              title="紧凑表格看板"
            >
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"/>
              </svg>
              <span>紧凑表格</span>
            </button>
            <button
              @click="setViewMode('cards')"
              class="px-2.5 py-1 text-xs font-medium rounded-lg transition-colors flex items-center gap-1.5"
              :class="viewMode === 'cards' ? 'bg-blue-600 text-white font-bold shadow-sm' : 'text-blue-200/80 hover:text-white'"
              title="卡片看板"
            >
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z"/>
              </svg>
              <span>卡片</span>
            </button>
          </div>

          <!-- 全局一键刷新配额 -->
          <button
            @click="refreshAllQuotas"
            :disabled="refreshingAll"
            class="btn btn-sm btn-primary flex items-center gap-1.5 shrink-0"
            title="同时向三大渠道官方发送请求拉取最新配额数据"
          >
            <svg class="w-3.5 h-3.5" :class="{ 'animate-spin': refreshingAll }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
            </svg>
            <span>{{ refreshingAll ? '刷新配额中...' : '一键刷新全部额度' }}</span>
          </button>
        </div>
      </div>

      <!-- Filters & Search Toolbar -->
      <div class="flex flex-col md:flex-row md:items-center md:justify-between gap-3 text-xs">
        <!-- 渠道切换标签页 -->
        <div class="flex items-center gap-1.5 overflow-x-auto pb-1 md:pb-0">
          <button
            v-for="ch in channelFilterTabs"
            :key="ch.value"
            @click="selectedChannel = ch.value"
            class="px-3 py-1.5 rounded-lg border font-medium transition-all shrink-0 flex items-center gap-1.5"
            :class="selectedChannel === ch.value
              ? 'bg-blue-600/20 border-blue-500/50 text-blue-200'
              : 'border-gray-800 bg-gray-900/60 text-gray-400 hover:text-white hover:border-gray-700'"
          >
            <span>{{ ch.icon }}</span>
            <span>{{ ch.label }}</span>
            <span class="text-[10px] px-1.5 py-0.2 rounded-full bg-gray-800/90 font-mono">{{ ch.count }}</span>
          </button>
        </div>

        <!-- 状态过滤、排序与搜索 -->
        <div class="flex flex-wrap items-center gap-2">
          <!-- 状态筛选 -->
          <select v-model="statusFilter" class="input input-sm text-xs py-1 px-2 w-auto">
            <option value="all">全部状态</option>
            <option value="active">正常 / 激活</option>
            <option value="error">受限 / 异常</option>
          </select>

          <!-- 排序 -->
          <select v-model="sortOrder" class="input input-sm text-xs py-1 px-2 w-auto">
            <option value="default">默认排序</option>
            <option value="channel">按渠道分组</option>
            <option value="quota_high">配额从多到少</option>
            <option value="quota_low">配额从少到多</option>
            <option value="email">按账号排序</option>
          </select>

          <!-- 搜索框 -->
          <div class="relative w-44 md:w-56">
            <input
              v-model="searchQuery"
              type="text"
              class="input input-sm w-full pl-7 text-xs"
              placeholder="搜索账号/邮箱/等级..."
            />
            <svg class="w-3.5 h-3.5 absolute left-2 top-2 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
            </svg>
          </div>
        </div>
      </div>

      <!-- Loading State -->
      <div v-if="loading" class="text-center py-16 text-gray-400">
        <svg class="w-8 h-8 animate-spin mx-auto text-blue-500 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
        </svg>
        加载多渠道额度数据中...
      </div>

      <!-- Empty State -->
      <div v-else-if="filteredUnifiedAccounts.length === 0" class="py-12 text-center text-gray-500 space-y-2">
        <div class="text-3xl">📭</div>
        <div class="text-sm text-gray-300">暂无匹配的账号额度信息</div>
        <p class="text-xs text-gray-500">可调整搜索词或过滤器，亦可前往对应渠道页面添加或接入新账号。</p>
      </div>

      <!-- View 1: 紧凑表格模式 (Compact Table) -->
      <div v-else-if="viewMode === 'table'" class="overflow-x-auto rounded-2xl border border-blue-500/30 bg-slate-900/90 shadow-2xl backdrop-blur-xl">
        <table class="w-full text-left text-xs">
          <thead class="border-b border-blue-500/25 bg-blue-500/15 text-blue-200 font-semibold tracking-wider text-[11px] uppercase">
            <tr>
              <th class="p-3.5 w-28">渠道</th>
              <th class="p-3.5 min-w-[200px]">账号身份</th>
              <th class="p-3.5 min-w-[220px]">核心额度水位监控</th>
              <th class="p-3.5 min-w-[150px]">次级指标 / 按量消耗</th>
              <th class="p-3.5 min-w-[130px]">重置 / 账单周期</th>
              <th class="p-3.5 w-24">状态</th>
              <th class="p-3.5 text-right min-w-[180px]">快捷操作</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-800/80">
            <tr
              v-for="item in paginatedUnifiedAccounts"
              :key="item.uniqueKey"
              class="hover:bg-slate-800/50 transition-colors"
              :class="{
                'bg-blue-500/15 hover:bg-blue-500/20 border-l-4 border-blue-400': item.channel === 'codex' && item.active,
                'bg-amber-500/15 hover:bg-amber-500/20 border-l-4 border-amber-400': item.channel === 'antigravity' && item.active,
                'bg-purple-500/15 hover:bg-purple-500/20 border-l-4 border-purple-400': item.channel === 'cursor' && item.active,
              }"
            >
              <!-- 渠道 -->
              <td class="p-3 whitespace-nowrap">
                <span class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-[11px] font-medium border" :class="item.channelBadgeClass">
                  <span>{{ item.channelIcon }}</span>
                  <span>{{ item.channelLabel }}</span>
                </span>
              </td>

              <!-- 账号身份 -->
              <td class="p-3">
                <div class="flex items-center gap-2.5">
                  <div class="h-8 w-8 rounded-lg flex items-center justify-center font-bold text-xs shrink-0 overflow-hidden border" :class="item.avatarClass">
                    <template v-if="privacyMode">🔒</template>
                    <template v-else-if="item.avatarUrl">
                      <img :src="item.avatarUrl" class="h-full w-full object-cover" />
                    </template>
                    <template v-else>
                      <span>{{ item.avatarInitial }}</span>
                    </template>
                  </div>
                  <div class="min-w-0">
                    <div class="flex items-center gap-1.5">
                      <span class="font-semibold text-white truncate max-w-[150px]" :title="privacyMode ? '已隐藏' : item.email">
                        {{ getUnifiedDisplayName(item) }}
                      </span>
                      <span class="px-1.5 py-0.2 text-[9px] font-mono rounded uppercase shrink-0 border" :class="item.tierBadgeClass">
                        {{ item.tier }}
                      </span>
                      <span v-if="item.active" class="px-1.5 py-0.2 text-[9px] rounded bg-emerald-500/20 text-emerald-400 border border-emerald-500/30 shrink-0 font-medium">
                        当前生效
                      </span>
                    </div>
                    <div v-if="!privacyMode && item.email && item.displayName && item.displayName !== item.email" class="text-[11px] text-gray-500 truncate max-w-[150px]">
                      {{ item.email }}
                    </div>
                    <div v-else-if="privacyMode" class="text-[10px] text-indigo-400/90 font-mono flex items-center gap-1">
                      <span>🔒 隐私脱敏</span>
                    </div>
                  </div>
                </div>
              </td>

              <!-- 核心额度水位监控 -->
              <td class="p-3">
                <!-- Antigravity Quota -->
                <template v-if="item.channel === 'antigravity'">
                  <div class="space-y-1.5 max-w-[200px]">
                    <div class="flex items-center justify-between text-[11px]">
                      <span class="text-gray-400">Claude (5h)</span>
                      <span :class="getQuotaColorClass(item.antigravityClaude5h)" class="font-mono font-bold">
                        {{ item.antigravityClaude5h != null ? `${item.antigravityClaude5h}%` : '--' }}
                      </span>
                    </div>
                    <div class="h-2 w-full rounded-full bg-slate-800 border border-slate-700/60 p-0.5 overflow-hidden">
                      <div
                        class="h-full rounded-full transition-all"
                        :class="getQuotaBarClass(item.antigravityClaude5h)"
                        :style="{ width: `${item.antigravityClaude5h || 0}%` }"
                      ></div>
                    </div>
                    <div class="flex items-center justify-between text-[10px] text-gray-500">
                      <span>预计重置: {{ formatResetTime(item.antigravityClaude5hReset, item.antigravityClaude5h) }}</span>
                    </div>
                  </div>
                </template>

                <!-- Cursor Quota -->
                <template v-else-if="item.channel === 'cursor'">
                  <div class="space-y-1.5 max-w-[200px]">
                    <div class="flex items-center justify-between text-[11px]">
                      <span class="text-gray-400">Fast 请求</span>
                      <div class="flex items-center gap-1 font-mono">
                        <span class="text-gray-400">{{ item.cursorPlanUsed }} / {{ item.cursorPlanLimit }}</span>
                        <span :class="getQuotaColorClass(item.cursorPlanPct)" class="font-bold">
                          {{ item.cursorPlanPct }}%
                        </span>
                      </div>
                    </div>
                    <div class="h-2 w-full rounded-full bg-slate-800 border border-slate-700/60 p-0.5 overflow-hidden">
                      <div
                        class="h-full rounded-full transition-all"
                        :class="getQuotaBarClass(item.cursorPlanPct)"
                        :style="{ width: `${Math.min(100, Math.max(0, item.cursorPlanPct))}%` }"
                      ></div>
                    </div>
                    <div class="text-[10px] text-gray-500">剩余 {{ item.cursorPlanRemaining }} 次快速请求</div>
                  </div>
                </template>

                <!-- Codex Quota -->
                <template v-else>
                  <div class="space-y-1 max-w-[200px]">
                    <div class="flex items-center justify-between text-[11px]">
                      <span class="text-gray-400">5h 限额剩余</span>
                      <span :class="getQuotaColorClass(item.codex5hPct)" class="font-mono font-bold">
                        {{ item.codex5hPct != null ? `${item.codex5hPct}%` : (item.raw.quota_is_forbidden ? '受限' : '未查询') }}
                      </span>
                    </div>
                    <div class="h-2 w-full rounded-full bg-slate-800 border border-slate-700/60 p-0.5 overflow-hidden">
                      <div
                        class="h-full rounded-full transition-all"
                        :class="getQuotaBarClass(item.codex5hPct)"
                        :style="{ width: `${item.codex5hPct != null ? item.codex5hPct : 100}%` }"
                      ></div>
                    </div>
                    <div class="flex items-center justify-between text-[10px] text-gray-500">
                      <span>{{ item.raw.account_type === 'api' ? (item.raw.model || 'API 直通') : `Plan: ${item.raw.plan || 'Plus'}` }}</span>
                      <span v-if="item.raw.in_proxy_pool || item.raw.proxy_enabled" class="text-blue-400">已入池</span>
                    </div>
                  </div>
                </template>
              </td>

              <!-- 次级指标 / 按量消耗 -->
              <td class="p-3 text-[11px]">
                <template v-if="item.channel === 'antigravity'">
                  <div class="space-y-0.5">
                    <div class="flex items-center justify-between text-gray-300">
                      <span>Claude (周):</span>
                      <span class="font-mono font-bold" :class="getQuotaColorClass(item.antigravityClaudeWeekly)">
                        {{ item.antigravityClaudeWeekly != null ? `${item.antigravityClaudeWeekly}%` : '--' }}
                      </span>
                    </div>
                    <div class="flex items-center justify-between text-gray-400 text-[10px]">
                      <span>Gemini (5h):</span>
                      <span class="font-mono">{{ item.antigravityGemini5h != null ? `${item.antigravityGemini5h}%` : '--' }}</span>
                    </div>
                  </div>
                </template>

                <template v-else-if="item.channel === 'cursor'">
                  <div class="space-y-0.5 font-mono">
                    <div class="flex items-center justify-between">
                      <span class="text-gray-400">超额按量:</span>
                      <span class="font-bold text-indigo-300">${{ (item.cursorOnDemandCents / 100).toFixed(2) }}</span>
                    </div>
                    <div class="flex items-center justify-between text-[10px] text-gray-500">
                      <span>Auto / API:</span>
                      <span>{{ item.raw.auto_percent_used || 0 }}% / {{ item.raw.api_percent_used || 0 }}%</span>
                    </div>
                  </div>
                </template>

                <template v-else>
                  <div class="space-y-0.5 text-gray-400 font-mono">
                    <div class="flex items-center justify-between">
                      <span>7d 限额剩余:</span>
                      <span :class="getQuotaColorClass(item.codex7dPct)">{{ item.codex7dPct != null ? `${item.codex7dPct}%` : '--' }}</span>
                    </div>
                    <div class="text-[10px] text-gray-500">
                      {{ item.raw.account_type === 'api' ? 'API 渠道' : 'OAuth 渠道' }}
                    </div>
                  </div>
                </template>
              </td>

              <!-- 重置 / 账单周期 -->
              <td class="p-3 text-[11px] text-gray-400">
                <template v-if="item.channel === 'cursor'">
                  <div>{{ formatBillingCycle(item.raw.billing_cycle_end) }}</div>
                  <div v-if="item.lastRefreshAt" class="text-[10px] text-gray-500">{{ formatRelativeTime(item.lastRefreshAt) }}</div>
                </template>
                <template v-else-if="item.channel === 'antigravity'">
                  <div>{{ formatResetTime(item.antigravityClaudeWeeklyReset, item.antigravityClaudeWeekly) }}</div>
                  <div v-if="item.lastRefreshAt" class="text-[10px] text-gray-500">{{ formatRelativeTime(item.lastRefreshAt) }}</div>
                </template>
                <template v-else>
                  <div>{{ item.raw.account_type === 'api' ? '持续计费' : '按月续费' }}</div>
                  <div v-if="item.lastRefreshAt" class="text-[10px] text-gray-500">{{ formatRelativeTime(item.lastRefreshAt) }}</div>
                </template>
              </td>

              <!-- 状态 -->
              <td class="p-3">
                <span v-if="item.statusType === 'success'" class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[11px] font-medium bg-emerald-500/15 text-emerald-300 border border-emerald-500/30">
                  <span class="h-1.5 w-1.5 rounded-full bg-emerald-400"></span> 正常
                </span>
                <span v-else-if="item.statusType === 'danger'" class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[11px] font-medium bg-rose-500/15 text-rose-300 border border-rose-500/30" :title="item.statusMessage">
                  <span class="h-1.5 w-1.5 rounded-full bg-rose-400"></span> 受限
                </span>
                <span v-else class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[11px] font-medium bg-amber-500/15 text-amber-300 border border-amber-500/30" :title="item.statusMessage">
                  <span class="h-1.5 w-1.5 rounded-full bg-amber-400"></span> 待查
                </span>
              </td>

              <!-- 快捷操作 -->
              <td class="p-3 text-right">
                <div class="flex items-center justify-end gap-1.5">
                  <button
                    @click="activateAccount(item)"
                    :disabled="item.active || activatingId === item.id"
                    class="btn btn-xs"
                    :class="item.active ? 'border border-emerald-500/30 bg-emerald-500/10 text-emerald-300 font-semibold cursor-default' : 'btn-primary'"
                    :title="item.active ? '当前生效中' : '设为当前渠道的主账号'"
                  >
                    <span v-if="activatingId === item.id" class="animate-spin text-[10px]">⏳</span>
                    <span v-else-if="item.active">✓ 生效中</span>
                    <span v-else>设为当前</span>
                  </button>

                  <button
                    @click="refreshAccount(item)"
                    :disabled="refreshingId === item.id"
                    class="btn btn-xs btn-secondary"
                    title="刷新该账号配额"
                  >
                    <svg class="w-3 h-3" :class="{ 'animate-spin': refreshingId === item.id }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
                    </svg>
                  </button>

                  <button
                    @click="goToChannel(item.channel)"
                    class="btn btn-xs btn-secondary"
                    title="前往对应渠道管理页"
                  >
                    <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"/>
                    </svg>
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- View 2: 卡片看板模式 (Cards Mode) -->
      <div v-else class="grid gap-3.5 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
        <div
          v-for="item in paginatedUnifiedAccounts"
          :key="item.uniqueKey"
          class="rounded-2xl border border-slate-700/70 bg-slate-900/90 p-4 space-y-3 transition-all duration-200 shadow-xl relative overflow-hidden backdrop-blur-xl hover:border-slate-600 flex flex-col justify-between"
          :class="{
            'ring-2 ring-blue-500/50 border-blue-500/50 bg-gradient-to-b from-blue-950/20 to-slate-900/90': item.channel === 'codex' && item.active,
            'ring-2 ring-amber-500/50 border-amber-500/50 bg-gradient-to-b from-amber-950/20 to-slate-900/90': item.channel === 'antigravity' && item.active,
            'ring-2 ring-purple-500/50 border-purple-500/50 bg-gradient-to-b from-purple-950/20 to-slate-900/90': item.channel === 'cursor' && item.active,
          }"
        >
          <!-- Card Header & Body Section -->
          <div class="space-y-2.5">
            <!-- Card Header -->
            <div class="flex items-center justify-between gap-2">
              <div class="flex items-center gap-2.5 min-w-0">
                <div class="h-9 w-9 rounded-xl flex items-center justify-center font-bold text-sm shrink-0 overflow-hidden border" :class="item.avatarClass">
                  <template v-if="privacyMode">🔒</template>
                  <template v-else-if="item.avatarUrl">
                    <img :src="item.avatarUrl" class="h-full w-full object-cover" />
                  </template>
                  <template v-else>
                    <span>{{ item.avatarInitial }}</span>
                  </template>
                </div>
                <div class="min-w-0">
                  <div class="flex items-center gap-1.5">
                    <span class="font-semibold text-white truncate max-w-[130px]" :title="privacyMode ? '已隐藏' : item.email">
                      {{ getUnifiedDisplayName(item) }}
                    </span>
                    <span class="px-1.5 py-0.2 text-[9px] font-mono rounded uppercase shrink-0 border" :class="item.tierBadgeClass">
                      {{ item.tier }}
                    </span>
                  </div>
                  <div class="flex items-center gap-1 mt-0.5">
                    <span class="text-[10px] font-medium px-1.5 py-0.2 rounded border" :class="item.channelBadgeClass">
                      {{ item.channelIcon }} {{ item.channelLabel }}
                    </span>
                    <span v-if="item.active" class="text-[10px] px-1.5 py-0.2 rounded bg-emerald-500/20 text-emerald-400 font-medium border border-emerald-500/30">
                      生效中
                    </span>
                  </div>
                </div>
              </div>

              <!-- Status indicator -->
              <div class="shrink-0 flex items-center gap-1.5">
                <span
                  v-if="privacyMode"
                  class="px-1.5 py-0.2 text-[9px] font-medium rounded-full bg-indigo-500/15 text-indigo-300 border border-indigo-500/25"
                  title="已开启隐私保护"
                >
                  🔒 隐私
                </span>
                <span v-if="item.statusType === 'success'" class="h-2 w-2 rounded-full bg-emerald-400 inline-block" title="状态正常"></span>
                <span v-else-if="item.statusType === 'danger'" class="h-2 w-2 rounded-full bg-rose-400 inline-block animate-pulse" :title="item.statusMessage || '状态受限/异常'"></span>
                <span v-else class="h-2 w-2 rounded-full bg-amber-400 inline-block" :title="item.statusMessage || '状态过期/待查'"></span>
              </div>
            </div>

            <!-- Status warning / error banner (基本报错一致: 统一大小、统一颜色) -->
            <div
              v-if="item.statusType === 'danger'"
              class="rounded-lg bg-rose-500/10 border border-rose-500/20 px-2.5 py-1.5 text-xs text-rose-400 flex items-center gap-1.5 min-h-[30px]"
              :title="item.statusMessage"
            >
              <span class="shrink-0 text-sm">⚠️</span>
              <span class="truncate font-medium">{{ item.statusMessage }}</span>
            </div>
            <div
              v-else-if="item.statusType === 'warning'"
              class="rounded-lg bg-amber-500/10 border border-amber-500/20 px-2.5 py-1.5 text-xs text-amber-300 flex items-center gap-1.5 min-h-[30px]"
              :title="item.statusMessage"
            >
              <span class="shrink-0 text-sm">⚠️</span>
              <span class="truncate font-medium">{{ item.statusMessage }}</span>
            </div>

            <!-- Card Body: Quota Indicator -->
            <div class="pt-2 border-t border-slate-800/80">
              <!-- Antigravity Quota Card Body -->
              <template v-if="item.channel === 'antigravity'">
                <div class="rounded-xl bg-slate-800/80 p-3 border border-slate-700/70 space-y-1.5 shadow-inner">
                  <div class="flex items-center justify-between text-xs">
                    <span class="text-gray-300 font-medium">Claude (5h)</span>
                    <span :class="item.statusType === 'danger' && item.antigravityClaude5h == null ? 'text-rose-400 font-bold' : getQuotaColorClass(item.antigravityClaude5h)" class="font-mono font-bold">
                      {{ item.statusType === 'danger' && item.antigravityClaude5h == null ? '受限 (403)' : (item.antigravityClaude5h != null ? `${item.antigravityClaude5h}%` : '--') }}
                    </span>
                  </div>
                  <div class="h-2 w-full rounded-full bg-slate-900 border border-slate-700/60 p-0.5 overflow-hidden">
                    <div
                      class="h-full rounded-full transition-all"
                      :class="item.statusType === 'danger' && !item.antigravityClaude5h ? 'bg-rose-500' : getQuotaBarClass(item.antigravityClaude5h)"
                      :style="{ width: `${item.antigravityClaude5h || 0}%` }"
                    ></div>
                  </div>
                  <div class="flex items-center justify-between text-[10px] text-gray-400">
                    <span>重置: {{ formatResetTime(item.antigravityClaude5hReset, item.antigravityClaude5h) }}</span>
                    <span>周: {{ item.antigravityClaudeWeekly != null ? `${item.antigravityClaudeWeekly}%` : '--' }}</span>
                  </div>
                </div>
              </template>

              <!-- Cursor Quota Card Body -->
              <template v-else-if="item.channel === 'cursor'">
                <div class="rounded-xl bg-slate-800/80 p-3 border border-slate-700/70 space-y-1.5 shadow-inner">
                  <div class="flex items-center justify-between text-xs">
                    <span class="text-gray-300 font-medium">Fast 请求</span>
                    <div class="flex items-center gap-1 font-mono">
                      <span class="text-gray-400">{{ item.cursorPlanUsed }} / {{ item.cursorPlanLimit }}</span>
                      <span :class="item.statusType === 'danger' && item.cursorPlanPct === 0 ? 'text-rose-400 font-bold' : getQuotaColorClass(item.cursorPlanPct)" class="font-bold">
                        {{ item.cursorPlanPct }}%
                      </span>
                    </div>
                  </div>
                  <div class="h-2 w-full rounded-full bg-slate-900 border border-slate-700/60 p-0.5 overflow-hidden">
                    <div
                      class="h-full rounded-full transition-all"
                      :class="item.statusType === 'danger' && !item.cursorPlanPct ? 'bg-rose-500' : getQuotaBarClass(item.cursorPlanPct)"
                      :style="{ width: `${Math.min(100, Math.max(0, item.cursorPlanPct))}%` }"
                    ></div>
                  </div>
                  <div class="flex items-center justify-between text-[10px] text-gray-400">
                    <span>剩余 {{ item.cursorPlanRemaining }} 次</span>
                    <span class="font-mono text-purple-300 font-bold">${{ (item.cursorOnDemandCents / 100).toFixed(2) }}</span>
                  </div>
                </div>
              </template>

              <!-- Codex Quota Card Body -->
              <template v-else>
                <div class="rounded-xl bg-slate-800/80 p-3 border border-slate-700/70 space-y-1.5 shadow-inner">
                  <div class="flex items-center justify-between text-xs">
                    <span class="text-gray-300 font-medium">5h 限额剩余</span>
                    <span :class="item.statusType === 'danger' && item.codex5hPct == null ? 'text-rose-400 font-bold' : getQuotaColorClass(item.codex5hPct)" class="font-mono font-bold">
                      {{ item.statusType === 'danger' && item.codex5hPct == null ? '受限 (403)' : (item.codex5hPct != null ? `${item.codex5hPct}%` : (item.raw.quota_is_forbidden ? '受限' : '未查询')) }}
                    </span>
                  </div>
                  <div class="h-2 w-full rounded-full bg-slate-900 border border-slate-700/60 p-0.5 overflow-hidden">
                    <div
                      class="h-full rounded-full transition-all"
                      :class="item.statusType === 'danger' && !item.codex5hPct ? 'bg-rose-500' : getQuotaBarClass(item.codex5hPct)"
                      :style="{ width: `${item.statusType === 'danger' && !item.codex5hPct ? 0 : (item.codex5hPct != null ? item.codex5hPct : 0)}%` }"
                    ></div>
                  </div>
                  <div class="flex items-center justify-between text-[10px] text-gray-400">
                    <span>{{ item.raw.account_type === 'api' ? 'API 账号' : `Plan: ${item.raw.plan || 'Plus'}` }}</span>
                    <span v-if="item.raw.in_proxy_pool || item.raw.proxy_enabled" class="text-blue-400 font-medium">已入池</span>
                    <span v-else class="text-gray-500">未入池</span>
                  </div>
                </div>
              </template>
            </div>
          </div>

          <!-- Card Actions -->
          <div class="flex items-center justify-between pt-2.5 border-t border-slate-800/80 gap-2">
            <button
              @click="activateAccount(item)"
              :disabled="item.active || activatingId === item.id"
              class="btn btn-xs flex-1"
              :class="item.active ? 'border border-emerald-500/30 bg-emerald-500/10 text-emerald-300 font-semibold cursor-default' : 'btn-primary'"
            >
              <span v-if="activatingId === item.id" class="animate-spin text-[10px]">⏳</span>
              <span v-else-if="item.active">✓ 生效中</span>
              <span v-else>设为当前</span>
            </button>

            <button
              @click="refreshAccount(item)"
              :disabled="refreshingId === item.id"
              class="btn btn-xs btn-secondary"
              title="刷新配额"
            >
              <svg class="w-3 h-3" :class="{ 'animate-spin': refreshingId === item.id }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
              </svg>
            </button>

            <button
              @click="goToChannel(item.channel)"
              class="btn btn-xs btn-secondary"
              title="进入渠道页面"
            >
              <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"/>
              </svg>
            </button>
          </div>
        </div>
      </div>

      <!-- Pagination -->
      <div v-if="totalPages > 1" class="flex flex-wrap items-center justify-between gap-3 pt-3 border-t border-gray-800 text-xs">
        <span class="text-gray-400">
          第 {{ currentPage }} / {{ totalPages }} 页 (显示 {{ paginationRangeText }})
        </span>
        <div class="flex items-center gap-2">
          <button
            class="btn btn-xs btn-secondary"
            :disabled="currentPage <= 1"
            @click="currentPage = Math.max(1, currentPage - 1)"
          >
            上一页
          </button>
          <button
            class="btn btn-xs btn-secondary"
            :disabled="currentPage >= totalPages"
            @click="currentPage = Math.min(totalPages, currentPage + 1)"
          >
            下一页
          </button>
        </div>
      </div>
    </section>

    <!-- 4. Relay & Codex Call Monitoring -->
    <section class="relay-panel">
      <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <div class="relay-panel__badge">
            <span>调用监控</span>
          </div>
          <h2 class="relay-panel__title">Codex 与 Relay 调用</h2>
          <p class="relay-panel__desc">
            <span>本地代理和 Relay 转发记录集中在这里，自动刷新。</span>
            <span :class="pool.codex_api_service ? 'relay-panel__status-ok' : 'relay-panel__status-warn'">
              Codex {{ pool.codex_api_service ? '已启用' : '未启用' }}
            </span>
            <span :class="relayUsage.upstream_configured ? 'relay-panel__status-ok' : 'relay-panel__status-warn'">
              · Relay {{ relayUsage.upstream_configured ? '已配置' : '未配置' }}
            </span>
            <span v-if="relayUsage.codex_injected" class="relay-panel__status-ok"> · 已注入</span>
          </p>
        </div>
        <div class="stable-action-row sm:justify-end">
          <button class="btn btn-secondary" @click="router.push('/codex')">Codex 配置</button>
          <button class="btn btn-secondary" @click="router.push('/relay')">Relay 配置</button>
        </div>
      </div>

      <div class="mt-5 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <div class="dashboard-mini-stat">
          <div class="relay-stat-card__label">Codex 最近调用</div>
          <div class="relay-stat-card__value">{{ recentCodexCalls.length }}</div>
          <div class="relay-stat-card__sub">{{ pool.codex_api_service ? '服务运行中' : '服务未启用' }}</div>
        </div>
        <div class="dashboard-mini-stat">
          <div class="relay-stat-card__label">Relay 请求</div>
          <div class="relay-stat-card__value">{{ relayUsage.usage?.request_count || 0 }}</div>
          <div class="relay-stat-card__sub">流式 {{ relayUsage.usage?.stream_count || 0 }}</div>
        </div>
        <div class="dashboard-mini-stat dashboard-mini-stat--highlight">
          <div class="relay-stat-card__label">总 Tokens</div>
          <div class="relay-stat-card__value">{{ formatTokens(relayUsage.usage?.total_tokens) }}</div>
          <div class="relay-stat-card__sub">
            输入 {{ formatTokens(relayUsage.usage?.input_tokens) }} / 输出 {{ formatTokens(relayUsage.usage?.output_tokens) }}
          </div>
        </div>
        <div class="dashboard-mini-stat">
          <div class="relay-stat-card__label">最近调用</div>
          <div class="relay-stat-card__value relay-stat-card__value--sm" :title="relayUsage.usage?.last_model || '-'">
            {{ relayUsage.usage?.last_model || '暂无' }}
          </div>
          <div class="relay-stat-card__sub">{{ formatRelayTime(relayUsage.usage?.last_request_at) }}</div>
        </div>
      </div>

      <div class="mt-5">
        <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
          <div>
            <h3 class="relay-panel__subtitle">最近调用明细</h3>
            <p class="relay-panel__hint mt-1">
              共 {{ combinedCallRows.length }} 条，Codex {{ recentCodexCalls.length }} 条，Relay {{ recentRelayCalls.length }} 条
            </p>
          </div>
          <div class="flex flex-wrap items-center justify-end gap-2">
            <span class="relay-panel__hint">每 10 秒刷新</span>
            <select
              v-if="combinedCallRows.length > 0 || callSourceFilter !== 'all'"
              v-model="callSourceFilter"
              class="input h-8 w-auto py-1 pl-2 pr-6 text-sm"
            >
              <option value="all">全部来源</option>
              <option value="codex">仅 Codex</option>
              <option value="relay">仅 Relay</option>
            </select>
            <button
              v-if="combinedCallRows.length > 0"
              class="btn btn-secondary py-1 text-sm h-8 px-3"
              @click="clearAllHistory"
            >
              清空
            </button>
          </div>
        </div>

        <div v-if="filteredCombinedCallRows.length === 0" class="relay-call-empty">
          <template v-if="combinedCallRows.length === 0">
            暂无调用记录。通过 Codex 本地服务或 Relay 发起对话后，记录会显示在这里。
          </template>
          <template v-else>
            暂无匹配该来源的调用记录。
          </template>
        </div>
        <div v-else class="relay-call-wrap overflow-x-auto">
          <table class="relay-call-table w-full min-w-[860px] text-left text-sm">
            <thead>
              <tr>
                <th>时间</th>
                <th>来源</th>
                <th>账号 / 上游</th>
                <th>模型</th>
                <th>类型</th>
                <th>状态 / 消耗</th>
                <th>详情</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="call in paginatedCombinedCalls" :key="call.id">
                <td class="cell-muted whitespace-nowrap">{{ formatRelayTime(call.timestamp) }}</td>
                <td class="whitespace-nowrap">
                  <span class="relay-type-tag" :class="call.sourceClass">{{ call.sourceLabel }}</span>
                </td>
                <td class="cell-accent whitespace-nowrap" :title="call.primary">
                  {{ call.primary }}
                </td>
                <td class="cell-text whitespace-nowrap" :title="call.model">
                  {{ call.model }}
                </td>
                <td class="whitespace-nowrap">
                  <span class="relay-type-tag">{{ call.typeLabel }}</span>
                </td>
                <td class="whitespace-nowrap" :title="call.statusTitle">
                  <span class="relay-type-tag" :class="call.statusClass">{{ call.statusLabel }}</span>
                </td>
                <td class="cell-secondary whitespace-nowrap" :title="call.detailTitle">
                  {{ call.detail }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-if="combinedTotalPages > 1" class="mt-4 flex items-center justify-center gap-2">
          <button class="btn btn-sm btn-secondary" :disabled="combinedCurrentPage === 1" @click="combinedCurrentPage--">上一页</button>
          <span class="text-sm text-gray-400">第 {{ combinedCurrentPage }} / {{ combinedTotalPages }} 页</span>
          <button class="btn btn-sm btn-secondary" :disabled="combinedCurrentPage === combinedTotalPages" @click="combinedCurrentPage++">下一页</button>
        </div>
      </div>
    </section>

    <!-- 5. System Runtime & Architecture -->
    <div class="grid min-h-0 flex-1 items-stretch gap-5 xl:grid-cols-[minmax(0,1.4fr)_360px]">
      <section class="card p-5 space-y-4">
        <div>
          <h2 class="text-lg font-semibold text-white">多渠道集成指南与架构</h2>
          <p class="mt-1 text-sm text-gray-400">EasyLLM 已深度整合三大渠道，赋能全场景 AI 开发体验。</p>
        </div>
        <div class="grid sm:grid-cols-3 gap-3 pt-1">
          <div class="rounded-xl border border-blue-500/20 bg-blue-500/5 p-4 space-y-2">
            <div class="text-sm font-semibold text-blue-300 flex items-center gap-1.5">
              <span>🤖</span>
              <span>OpenAI / Codex</span>
            </div>
            <p class="text-xs text-gray-400 leading-relaxed">
              支持 OAuth 与 API 账号轮询池，无缝支持 GPT-5 / 6 系列模型映射与本地直通代理。
            </p>
            <div class="pt-1">
              <button @click="router.push('/codex')" class="text-xs text-blue-400 hover:underline">管理 Codex 账号 →</button>
            </div>
          </div>

          <div class="rounded-xl border border-amber-500/20 bg-amber-500/5 p-4 space-y-2">
            <div class="text-sm font-semibold text-amber-300 flex items-center gap-1.5">
              <span>🚀</span>
              <span>Google Antigravity</span>
            </div>
            <p class="text-xs text-gray-400 leading-relaxed">
              直连 Google 官方 OAuth，支持 Claude 与 Gemini 5小时/每周动态滑动窗口配额监控与一键唤醒。
            </p>
            <div class="pt-1">
              <button @click="router.push('/antigravity')" class="text-xs text-amber-400 hover:underline">管理 Antigravity →</button>
            </div>
          </div>

          <div class="rounded-xl border border-purple-500/20 bg-purple-500/5 p-4 space-y-2">
            <div class="text-sm font-semibold text-purple-300 flex items-center gap-1.5">
              <span>⚡</span>
              <span>Cursor IDE 接入</span>
            </div>
            <p class="text-xs text-gray-400 leading-relaxed">
              一键读取本机 Cursor IDE 登录会话，实时统计 Included Fast 请求剩余与超额 On-Demand 账单消耗。
            </p>
            <div class="pt-1">
              <button @click="router.push('/cursor')" class="text-xs text-purple-400 hover:underline">管理 Cursor →</button>
            </div>
          </div>
        </div>
      </section>

      <section class="space-y-4">
        <article class="card runtime-status-card p-5">
          <div>
            <h2 class="text-lg font-semibold text-white">运行状态</h2>
            <p class="mt-1 text-sm text-gray-500">来自本机 Go 进程的实时环境信息。</p>
          </div>
          <dl class="runtime-status-list">
            <div class="runtime-status-row">
              <dt>版本</dt>
              <dd class="runtime-status-value">v{{ sysInfo.version || '-' }}</dd>
            </div>
            <div class="runtime-status-row">
              <dt>运行时间</dt>
              <dd class="runtime-status-value">{{ sysInfo.uptime || '-' }}</dd>
            </div>
            <div class="runtime-status-row">
              <dt>数据库</dt>
              <dd class="runtime-status-value">{{ sysInfo.db_type || '-' }}</dd>
            </div>
            <div class="runtime-status-row">
              <dt>端口</dt>
              <dd class="runtime-status-value">{{ sysInfo.server_port || 8022 }}</dd>
            </div>
            <div class="runtime-status-row">
              <dt>Goroutines</dt>
              <dd class="runtime-status-value">{{ sysInfo.goroutines || '-' }}</dd>
            </div>
            <div class="runtime-status-row">
              <dt>内存占用</dt>
              <dd class="runtime-status-value">{{ sysInfo.memory_alloc_mb || '-' }} MB</dd>
            </div>
          </dl>
        </article>
      </section>
    </div>
  </div>
</template>

<script setup>
import { computed, inject, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import api, { openaiAPI, relayAPI, settingsAPI, antigravityAPI, cursorAPI, longApi } from '@/api'
import { filterAPIAccounts, filterOAuthAccounts } from '@/lib/accounts'

const notify = inject('notify', (msg, type) => alert(msg))
const confirmOperation = inject('confirmOperation')
const router = useRouter()

const loading = ref(false)
const refreshingAll = ref(false)
const refreshingId = ref(null)
const activatingId = ref(null)

// 3 Channels Data
const codexAccounts = ref([])
const antigravityAccounts = ref([])
const cursorAccounts = ref([])

const pool = ref({})
const sysInfo = ref({})
const relayUsage = ref({ usage: {}, recent_calls: [] })
const codexCalls = ref({ recent_calls: [] })
let relayRefreshTimer = null

const nowTimestamp = ref(Date.now())
let nowTimer = null

// View state
const privacyMode = ref(localStorage.getItem('easyllm.dashboard.privacyMode') === 'true')
const viewMode = ref(localStorage.getItem('easyllm.dashboard.viewMode') || 'table')
const selectedChannel = ref('all') // 'all', 'codex', 'antigravity', 'cursor'
const statusFilter = ref('all') // 'all', 'active', 'error'
const sortOrder = ref('default') // 'default', 'channel', 'quota_high', 'quota_low', 'email'
const searchQuery = ref('')
const currentPage = ref(1)
const PAGE_SIZE = 15

function setViewMode(mode) {
  viewMode.value = mode
  localStorage.setItem('easyllm.dashboard.viewMode', mode)
}

function togglePrivacyMode() {
  privacyMode.value = !privacyMode.value
  const flag = privacyMode.value ? 'true' : 'false'
  localStorage.setItem('easyllm.dashboard.privacyMode', flag)
  localStorage.setItem('easyllm.antigravity.privacyMode', flag)
  localStorage.setItem('easyllm.cursor.privacyMode', flag)
  notify(privacyMode.value ? '已开启一键隐私模式：全局脱敏隐藏账号ID、邮箱与敏感信息' : '已关闭隐私模式：正常显示账号信息', 'info')
}

// Sub-computations for Codex
const oauthAccounts = computed(() => filterOAuthAccounts(codexAccounts.value))
const apiAccounts = computed(() => filterAPIAccounts(codexAccounts.value))
const activeOAuthCount = computed(() => oauthAccounts.value.filter((a) => a.is_codex_active).length)
const activeAPICount = computed(() => apiAccounts.value.filter((a) => a.is_codex_active).length)

// Sub-computations for Antigravity
const activeAntigravityAccount = computed(() => antigravityAccounts.value.find((a) => a.active))
const avgAntigravityClaude = computed(() => {
  const pcts = antigravityAccounts.value
    .map(a => getBucketPercentage(a, 'claude:5h'))
    .filter(p => p !== null && p !== undefined)
  if (pcts.length === 0) return null
  return Math.round(pcts.reduce((a, b) => a + b, 0) / pcts.length)
})
const avgAntigravityGemini = computed(() => {
  const pcts = antigravityAccounts.value
    .map(a => getBucketPercentage(a, 'gemini:5h'))
    .filter(p => p !== null && p !== undefined)
  if (pcts.length === 0) return null
  return Math.round(pcts.reduce((a, b) => a + b, 0) / pcts.length)
})

// Sub-computations for Cursor
const activeCursorAccount = computed(() => cursorAccounts.value.find((a) => a.active))
const totalCursorFastRemaining = computed(() => {
  return cursorAccounts.value.reduce((s, a) => s + (a.plan_remaining || 0), 0)
})
const totalCursorOnDemandSpend = computed(() => {
  const cents = cursorAccounts.value.reduce((s, a) => s + (a.on_demand_used_cents || 0), 0)
  return (cents / 100).toFixed(2)
})

// ── Antigravity Quota Parsing Helpers ─────────────────────────
const BUCKET_DEFINITIONS = [
  { key: 'claude:5h', label: 'Claude (5h)', matches: ['3p-5h', 'claude:5h'] },
  { key: 'claude:weekly', label: 'Claude (周)', matches: ['3p-weekly', 'claude:weekly'] },
  { key: 'gemini:5h', label: 'Gemini (5h)', matches: ['gemini-5h', 'gemini:5h'] },
  { key: 'gemini:weekly', label: 'Gemini (周)', matches: ['gemini-weekly', 'gemini:weekly'] },
]

function getParsedQuotaModels(acc) {
  if (!acc.quota_models_json) return []
  try {
    const arr = JSON.parse(acc.quota_models_json)
    return Array.isArray(arr) ? arr : []
  } catch {
    return []
  }
}

function getCoreBuckets(acc) {
  const models = getParsedQuotaModels(acc)
  return BUCKET_DEFINITIONS.map(def => {
    const found = models.find(m => def.matches.includes(m.name))
    return {
      key: def.key,
      label: def.label,
      percentage: found ? found.percentage : null,
      resetTime: found?.reset_time || '',
    }
  })
}

function getBucketPercentage(acc, key) {
  const b = getCoreBuckets(acc).find(x => x.key === key)
  return b ? b.percentage : null
}

function getBucketResetTime(acc, key) {
  const b = getCoreBuckets(acc).find(x => x.key === key)
  return b ? b.resetTime : ''
}

// ── Unified Accounts Normalization ────────────────────────────
const unifiedAccounts = computed(() => {
  const list = []

  // 1. Codex accounts
  codexAccounts.value.forEach(acc => {
    const isOAuth = acc.account_type !== 'api'
    const pct5h = acc.quota_5h_used_percent != null ? Math.round(100 - acc.quota_5h_used_percent) : null
    const pct7d = acc.quota_7d_used_percent != null ? Math.round(100 - acc.quota_7d_used_percent) : null
    const isForbidden = !!acc.quota_is_forbidden
    const isReauth = acc.status === 'reauth_required'
    const isExpired = !!(acc.expires_at && new Date(acc.expires_at).getTime() < Date.now())
    const isExhausted = (pct5h !== null && pct5h <= 0) || (pct7d !== null && pct7d <= 0)
    const hasHttpErr = !!(acc._quota_http_status && acc._quota_http_status >= 400 && acc._quota_http_status !== 401 && acc._quota_http_status !== 403)

    let statusType = 'success'
    let statusMessage = '正常'
    if (isForbidden) {
      statusType = 'danger'
      statusMessage = '403 Forbidden: 账号受限或被风控'
    } else if (isReauth) {
      statusType = 'danger'
      statusMessage = '登录已失效，需重新授权登录'
    } else if (hasHttpErr) {
      statusType = 'danger'
      statusMessage = `配额查询异常 (HTTP ${acc._quota_http_status})`
    } else if (isExpired) {
      statusType = 'warning'
      statusMessage = 'Token 已过期，请刷新'
    } else if (isExhausted) {
      statusType = 'warning'
      statusMessage = '额度已用尽 (5h / 7d)'
    }

    list.push({
      uniqueKey: `codex-${acc.id}`,
      id: acc.id,
      channel: 'codex',
      channelLabel: 'Codex',
      channelIcon: '🤖',
      channelBadgeClass: 'border-blue-500/30 bg-blue-500/10 text-blue-300',
      avatarClass: 'bg-gradient-to-br from-blue-600 to-indigo-700 text-white border-blue-500/30',
      avatarInitial: (acc.email || acc.model_provider || 'C').slice(0, 1).toUpperCase(),
      avatarUrl: null,
      email: acc.email || '',
      displayName: acc.email || acc.model_provider || 'Codex 账号',
      tier: isOAuth ? (acc.plan || 'OAuth') : 'API',
      tierBadgeClass: isOAuth ? 'bg-blue-500/15 text-blue-300 border-blue-500/25' : 'bg-gray-700/60 text-gray-300 border-gray-600/30',
      active: !!acc.is_codex_active,
      statusType,
      statusMessage,
      lastRefreshAt: acc.updated_at || acc.created_at || '',
      // quota indicators
      primaryPct: pct5h,
      codex5hPct: pct5h,
      codex7dPct: pct7d,
      raw: acc,
    })
  })

  // 2. Antigravity accounts
  antigravityAccounts.value.forEach(acc => {
    const claude5h = getBucketPercentage(acc, 'claude:5h')
    const claudeWeekly = getBucketPercentage(acc, 'claude:weekly')
    const gemini5h = getBucketPercentage(acc, 'gemini:5h')
    const isForbidden = acc.status === 'forbidden'
    const isExpired = acc.status === 'expired'
    const isError = acc.status === 'error'

    let statusType = 'success'
    let statusMessage = '正常'
    if (isForbidden) {
      statusType = 'danger'
      statusMessage = '403 Forbidden: 账号无访问权限或受限'
    } else if (isError) {
      statusType = 'danger'
      statusMessage = acc.status_message ? `配额拉取异常: ${acc.status_message}` : '配额拉取异常'
    } else if (isExpired) {
      statusType = 'warning'
      statusMessage = 'Token 已过期，请刷新或重新授权'
    }

    list.push({
      uniqueKey: `antigravity-${acc.id}`,
      id: acc.id,
      channel: 'antigravity',
      channelLabel: 'Antigravity',
      channelIcon: '🚀',
      channelBadgeClass: 'border-amber-500/30 bg-amber-500/10 text-amber-300',
      avatarClass: 'bg-gradient-to-br from-amber-600 to-orange-700 text-white border-amber-500/30',
      avatarInitial: (acc.display_name || acc.email || 'A').slice(0, 1).toUpperCase(),
      avatarUrl: acc.picture,
      email: acc.email || '',
      displayName: acc.display_name || acc.email || 'Antigravity 账号',
      tier: acc.subscription_tier || 'Pro',
      tierBadgeClass: 'bg-amber-500/15 text-amber-300 border-amber-500/25',
      active: !!acc.active,
      statusType,
      statusMessage,
      lastRefreshAt: acc.last_refresh_at || '',
      // quota indicators
      primaryPct: claude5h,
      antigravityClaude5h: claude5h,
      antigravityClaude5hReset: getBucketResetTime(acc, 'claude:5h'),
      antigravityClaudeWeekly: claudeWeekly,
      antigravityClaudeWeeklyReset: getBucketResetTime(acc, 'claude:weekly'),
      antigravityGemini5h: gemini5h,
      raw: acc,
    })
  })

  // 3. Cursor accounts
  cursorAccounts.value.forEach(acc => {
    const isForbidden = acc.status === 'forbidden'
    const isExpired = acc.status === 'expired'
    const isError = acc.status === 'error'

    let statusType = 'success'
    let statusMessage = '正常'
    if (isForbidden) {
      statusType = 'danger'
      statusMessage = '403 Forbidden: 账号无访问权限或已被风控'
    } else if (isError) {
      statusType = 'danger'
      statusMessage = acc.status_message ? `配额拉取异常: ${acc.status_message}` : '配额拉取异常'
    } else if (isExpired) {
      statusType = 'warning'
      statusMessage = 'Token 已过期或失效，请刷新或重新配置'
    }

    list.push({
      uniqueKey: `cursor-${acc.id}`,
      id: acc.id,
      channel: 'cursor',
      channelLabel: 'Cursor',
      channelIcon: '⚡',
      channelBadgeClass: 'border-purple-500/30 bg-purple-500/10 text-purple-300',
      avatarClass: 'bg-gradient-to-br from-purple-600 to-indigo-800 text-white border-purple-500/30',
      avatarInitial: (acc.display_name || acc.email || 'C').slice(0, 1).toUpperCase(),
      avatarUrl: null,
      email: acc.email || '',
      displayName: acc.display_name || acc.email || 'Cursor 账号',
      tier: (acc.membership_type || 'FREE').toUpperCase(),
      tierBadgeClass: 'bg-purple-500/15 text-purple-300 border-purple-500/25',
      active: !!acc.active,
      statusType,
      statusMessage,
      lastRefreshAt: acc.last_refresh_at || '',
      // quota indicators
      primaryPct: acc.plan_percentage,
      cursorPlanPct: acc.plan_percentage,
      cursorPlanUsed: acc.plan_used,
      cursorPlanLimit: acc.plan_limit,
      cursorPlanRemaining: acc.plan_remaining,
      cursorOnDemandCents: acc.on_demand_used_cents || 0,
      raw: acc,
    })
  })

  return list
})

// ── Filter and Search ─────────────────────────────────────────
const channelFilterTabs = computed(() => [
  { value: 'all', label: '全部渠道', icon: '🌐', count: unifiedAccounts.value.length },
  { value: 'codex', label: 'Codex', icon: '🤖', count: codexAccounts.value.length },
  { value: 'antigravity', label: 'Antigravity', icon: '🚀', count: antigravityAccounts.value.length },
  { value: 'cursor', label: 'Cursor', icon: '⚡', count: cursorAccounts.value.length },
])

const filteredUnifiedAccounts = computed(() => {
  let list = unifiedAccounts.value.slice()

  // Channel filter
  if (selectedChannel.value !== 'all') {
    list = list.filter(item => item.channel === selectedChannel.value)
  }

  // Status filter
  if (statusFilter.value === 'active') {
    list = list.filter(item => item.statusType === 'success')
  } else if (statusFilter.value === 'error') {
    list = list.filter(item => item.statusType !== 'success')
  }

  // Search filter
  if (searchQuery.value.trim()) {
    const q = searchQuery.value.trim().toLowerCase()
    list = list.filter(item =>
      item.email.toLowerCase().includes(q) ||
      item.displayName.toLowerCase().includes(q) ||
      item.tier.toLowerCase().includes(q) ||
      item.channel.toLowerCase().includes(q)
    )
  }

  // Sort
  if (sortOrder.value === 'channel') {
    const order = { codex: 1, antigravity: 2, cursor: 3 }
    list.sort((a, b) => (order[a.channel] || 9) - (order[b.channel] || 9))
  } else if (sortOrder.value === 'quota_high') {
    list.sort((a, b) => (b.primaryPct || 0) - (a.primaryPct || 0))
  } else if (sortOrder.value === 'quota_low') {
    list.sort((a, b) => (a.primaryPct || 0) - (b.primaryPct || 0))
  } else if (sortOrder.value === 'email') {
    list.sort((a, b) => (a.email || '').localeCompare(b.email || ''))
  } else {
    // Default: active accounts first, then channel grouped
    list.sort((a, b) => {
      if (a.active !== b.active) return a.active ? -1 : 1
      return 0
    })
  }

  return list
})

const totalPages = computed(() => Math.ceil(filteredUnifiedAccounts.value.length / PAGE_SIZE) || 1)

const paginatedUnifiedAccounts = computed(() => {
  const start = (currentPage.value - 1) * PAGE_SIZE
  return filteredUnifiedAccounts.value.slice(start, start + PAGE_SIZE)
})

const paginationRangeText = computed(() => {
  if (filteredUnifiedAccounts.value.length === 0) return '0 个账号'
  const start = (currentPage.value - 1) * PAGE_SIZE + 1
  const end = Math.min(filteredUnifiedAccounts.value.length, start + PAGE_SIZE - 1)
  return `${start}-${end} / 共 ${filteredUnifiedAccounts.value.length} 个账号`
})

watch([selectedChannel, statusFilter, searchQuery], () => {
  currentPage.value = 1
})

function getUnifiedDisplayName(item) {
  if (privacyMode.value) {
    const list = unifiedAccounts.value.filter(a => a.channel === item.channel)
    const idx = list.findIndex(a => a.id === item.id)
    return `${item.channelLabel} #${idx >= 0 ? idx + 1 : (item.id || '').slice(0, 4)}`
  }
  return item.displayName
}

// ── Quota Action Handlers ─────────────────────────────────────
async function activateAccount(item) {
  activatingId.value = item.id
  try {
    if (item.channel === 'codex') {
      await api.post(`/openai/accounts/${item.id}/switch`)
      notify(`已切换 Codex 当前生效账号为「${getUnifiedDisplayName(item)}」`, 'success')
    } else if (item.channel === 'antigravity') {
      const res = await antigravityAPI.activate(item.id)
      notify(
        res.message || `已激活 Antigravity 账号「${getUnifiedDisplayName(item)}」`,
        res.sync_warning ? 'warning' : 'success'
      )
    } else if (item.channel === 'cursor') {
      await cursorAPI.activate(item.id)
      notify(`已激活 Cursor 账号「${getUnifiedDisplayName(item)}」并同步写回 IDE`, 'success')
    }
    await loadDashboard()
  } catch (err) {
    notify(`激活失败: ${err.message}`, 'error')
  } finally {
    activatingId.value = null
  }
}

async function refreshAccount(item) {
  refreshingId.value = item.id
  try {
    if (item.channel === 'codex') {
      await longApi.post('/openai/accounts/fetch-quotas', { ids: [item.id] })
      notify(`Codex 账号「${getUnifiedDisplayName(item)}」配额已更新`, 'success')
    } else if (item.channel === 'antigravity') {
      await antigravityAPI.refresh(item.id)
      notify(`Antigravity 账号「${getUnifiedDisplayName(item)}」配额已更新`, 'success')
    } else if (item.channel === 'cursor') {
      await cursorAPI.refresh(item.id)
      notify(`Cursor 账号「${getUnifiedDisplayName(item)}」配额已更新`, 'success')
    }
    await loadDashboard()
  } catch (err) {
    notify(`刷新配额失败: ${err.message}`, 'error')
  } finally {
    refreshingId.value = null
  }
}

function goToChannel(channel) {
  if (channel === 'codex') router.push('/codex')
  else if (channel === 'antigravity') router.push('/antigravity')
  else if (channel === 'cursor') router.push('/cursor')
}

async function refreshAllQuotas() {
  refreshingAll.value = true
  try {
    const promises = [
      longApi.post('/openai/accounts/fetch-quotas', { ids: [] }).catch(e => console.warn('Codex refresh failed:', e)),
      antigravityAPI.refreshAll().catch(e => console.warn('Antigravity refresh failed:', e)),
      cursorAPI.refreshAll().catch(e => console.warn('Cursor refresh failed:', e)),
    ]
    await Promise.all(promises)
    notify('全渠道配额刷新已触发完成！', 'success')
    await loadDashboard()
  } catch (err) {
    notify('全渠道配额刷新失败: ' + err.message, 'error')
  } finally {
    refreshingAll.value = false
  }
}

// ── Helpers ───────────────────────────────────────────────────
function getQuotaColorClass(pct) {
  if (pct === null || pct === undefined) return 'text-gray-500'
  if (pct < 20) return 'text-rose-400 font-bold'
  if (pct < 50) return 'text-amber-400 font-semibold'
  if (pct < 80) return 'text-teal-400'
  return 'text-emerald-400 font-bold'
}

function getQuotaBarClass(pct) {
  if (pct === null || pct === undefined) return 'bg-gray-700'
  if (pct < 20) return 'bg-rose-500'
  if (pct < 50) return 'bg-amber-500'
  if (pct < 80) return 'bg-teal-500'
  return 'bg-emerald-500'
}

function formatRelativeTime(isoStr) {
  if (!isoStr) return '-'
  try {
    const d = new Date(isoStr)
    const now = new Date()
    const diffSec = Math.floor((now - d) / 1000)
    if (diffSec < 60) return '刚刚'
    if (diffSec < 3600) return `${Math.floor(diffSec / 60)} 分钟前`
    return `${d.getMonth() + 1}/${d.getDate()} ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
  } catch (e) {
    return isoStr
  }
}

function formatBillingCycle(endDateStr) {
  if (!endDateStr) return '周期计费'
  try {
    const d = new Date(endDateStr)
    const now = new Date()
    const diffDays = Math.ceil((d - now) / (1000 * 60 * 60 * 24))
    if (diffDays > 0) return `${diffDays}天后重置`
    return endDateStr
  } catch (e) {
    return endDateStr
  }
}

function formatResetTime(isoStr, percentage) {
  if (percentage !== null && percentage !== undefined && percentage >= 100) {
    return '额度满额'
  }
  if (!isoStr) return '实时周期'
  try {
    const d = new Date(isoStr)
    const diffMs = d.getTime() - nowTimestamp.value
    if (diffMs <= 0) return '待刷新'
    const diffMin = Math.round(diffMs / 60000)
    const hours = Math.floor(diffMin / 60)
    const mins = diffMin % 60
    if (diffMin < 60) return `${diffMin}分钟后`
    if (hours < 24) return `${hours}h ${mins}m 后`
    const days = Math.floor(hours / 24)
    return `${days}天 ${hours % 24}h 后`
  } catch {
    return '周期计算中'
  }
}

// ── Relay Calls & Existing Call Section Data ───────────────────
const recentRelayCalls = computed(() => {
  const calls = relayUsage.value?.recent_calls
  return Array.isArray(calls) ? calls : []
})
const recentCodexCalls = computed(() => {
  const calls = codexCalls.value?.recent_calls
  return Array.isArray(calls) ? calls : []
})

const callSourceFilter = ref('all')
const combinedCurrentPage = ref(1)
const COMBINED_PAGE_SIZE = 10

const combinedCallRows = computed(() => {
  const codexRows = recentCodexCalls.value.map((call, index) => ({
    id: `codex-${call.timestamp || index}-${index}`,
    source: 'codex',
    sourceLabel: 'Codex',
    sourceClass: 'relay-type-tag--codex',
    timestamp: call.timestamp,
    primary: call.account_email || call.account_id || '-',
    model: call.model || '-',
    typeLabel: formatCodexCallType(call),
    statusLabel: call.status_code || '-',
    statusClass: statusClass(call.status_code),
    statusTitle: call.error || '',
    detail: `${call.path || '-'} · ${formatDuration(call.duration_ms)}`,
    detailTitle: `${call.path || '-'} · ${formatDuration(call.duration_ms)}`,
  }))

  const relayRows = recentRelayCalls.value.map((call, index) => {
    const provider = call.provider || relayUsage.value.upstream_label || '-'
    const codexModel = call.codex_model || '-'
    const upstreamModel = call.upstream_model || '-'
    const model = upstreamModel !== '-' ? `${codexModel} -> ${upstreamModel}` : codexModel
    const tokenDetail = `输入 ${formatTokens(call.input_tokens)} / 输出 ${formatTokens(call.output_tokens)}`
    return {
      id: `relay-${call.timestamp || index}-${index}`,
      source: 'relay',
      sourceLabel: 'Relay',
      sourceClass: 'relay-type-tag--relay',
      timestamp: call.timestamp,
      primary: provider,
      model,
      typeLabel: call.stream ? '流式' : '非流式',
      statusLabel: formatTokens(call.total_tokens),
      statusClass: 'relay-type-tag--token',
      statusTitle: tokenDetail,
      detail: tokenDetail,
      detailTitle: tokenDetail,
    }
  })

  return [...codexRows, ...relayRows].sort((a, b) => {
    const at = new Date(a.timestamp || 0).getTime()
    const bt = new Date(b.timestamp || 0).getTime()
    return (Number.isNaN(bt) ? 0 : bt) - (Number.isNaN(at) ? 0 : at)
  })
})

const filteredCombinedCallRows = computed(() => {
  if (callSourceFilter.value === 'all') return combinedCallRows.value
  return combinedCallRows.value.filter((call) => call.source === callSourceFilter.value)
})

const combinedTotalPages = computed(() => Math.ceil(filteredCombinedCallRows.value.length / COMBINED_PAGE_SIZE) || 1)

const paginatedCombinedCalls = computed(() => {
  const start = (combinedCurrentPage.value - 1) * COMBINED_PAGE_SIZE
  return filteredCombinedCallRows.value.slice(start, start + COMBINED_PAGE_SIZE)
})

watch(callSourceFilter, () => {
  combinedCurrentPage.value = 1
})

watch(combinedTotalPages, (totalPages) => {
  if (combinedCurrentPage.value > totalPages) {
    combinedCurrentPage.value = totalPages
  }
})

async function clearAllHistory() {
  const confirmed = await requestOperationConfirm({
    title: '清空调用记录',
    message: '确认清空最近的调用记录吗？',
    details: '会同时清空 Relay 调用历史和 Codex 本地服务调用记录。',
    confirmText: '清空记录',
    tone: 'danger',
  })
  if (!confirmed) return
  try {
    await Promise.all([
      api.delete('/relay/usage/history'),
      openaiAPI.clearCodexCalls(),
    ])
    await refreshCallSections()
    notify('调用记录已清空')
  } catch (err) {
    console.error('Failed to clear call history:', err)
  }
}

async function requestOperationConfirm(options) {
  if (typeof confirmOperation !== 'function') return false
  return await confirmOperation(options)
}

// ── Lifecycle & Data Loading ─────────────────────────────────
let dashboardVisibilityHandler = null

onMounted(async () => {
  await loadDashboard()
  relayRefreshTimer = setInterval(() => {
    if (document.hidden || window.__easyllm_hidden) return
    refreshCallSections({ includePool: true })
  }, 10000)
  nowTimer = setInterval(() => {
    if (document.hidden || window.__easyllm_hidden) return
    nowTimestamp.value = Date.now()
  }, 10000)

  dashboardVisibilityHandler = () => {
    if (!document.hidden && !window.__easyllm_hidden) {
      nowTimestamp.value = Date.now()
      refreshCallSections({ includePool: true })
    }
  }
  document.addEventListener('visibilitychange', dashboardVisibilityHandler)
  window.addEventListener('easyllm-app-visibility', dashboardVisibilityHandler)
})

onUnmounted(() => {
  if (relayRefreshTimer) clearInterval(relayRefreshTimer)
  if (nowTimer) clearInterval(nowTimer)
  if (dashboardVisibilityHandler) {
    document.removeEventListener('visibilitychange', dashboardVisibilityHandler)
    window.removeEventListener('easyllm-app-visibility', dashboardVisibilityHandler)
  }
})

async function loadRelayUsage() {
  try {
    const relayData = await relayAPI.getUsage()
    relayUsage.value = relayData || { usage: {}, recent_calls: [] }
  } catch (error) {
    console.error('Failed to load relay usage:', error)
  }
}

async function loadCodexCalls() {
  try {
    const data = await openaiAPI.getCodexCalls()
    codexCalls.value = data || { recent_calls: [] }
  } catch (error) {
    console.error('Failed to load Codex calls:', error)
  }
}

async function loadPoolConfig() {
  try {
    const data = await api.get('/openai/service-config')
    pool.value = data || {}
  } catch (error) {
    console.error('Failed to load service config:', error)
  }
}

async function refreshCallSections(options = {}) {
  const tasks = [loadRelayUsage(), loadCodexCalls()]
  if (options.includePool) {
    tasks.push(loadPoolConfig())
  }
  await Promise.all(tasks)
}

async function loadDashboard() {
  loading.value = true
  try {
    const [accountData, poolData, systemData, agyData, curData] = await Promise.all([
      openaiAPI.list().catch(() => []),
      api.get('/openai/service-config').catch(() => ({})),
      settingsAPI.systemInfo().catch(() => ({})),
      antigravityAPI.list().catch(() => []),
      cursorAPI.list().catch(() => []),
    ])
    codexAccounts.value = Array.isArray(accountData) ? accountData : []
    pool.value = poolData || {}
    sysInfo.value = systemData || {}
    antigravityAccounts.value = Array.isArray(agyData) ? agyData : []
    cursorAccounts.value = Array.isArray(curData) ? curData : []
    await refreshCallSections()
  } catch (error) {
    notify?.(error.message || '加载总览失败', 'error')
  } finally {
    loading.value = false
  }
}

function formatTokens(value) {
  const n = Number(value) || 0
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(2)}M`
  if (n >= 10_000) return `${(n / 1_000).toFixed(1)}K`
  return n.toLocaleString()
}

function formatRelayTime(iso) {
  if (!iso) return '暂无记录'
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return iso
  return date.toLocaleString()
}

function formatDuration(value) {
  const ms = Number(value) || 0
  if (ms >= 1000) return `${(ms / 1000).toFixed(1)}s`
  return `${ms}ms`
}

function formatCodexCallType(call) {
  if (call?.path?.includes('/ws')) return 'WebSocket'
  if (call?.passthrough) return '直通'
  return call?.stream ? '流式' : '非流式'
}

function statusClass(status) {
  const code = Number(status) || 0
  if (code >= 200 && code < 400) return 'relay-type-tag--ok'
  if (code >= 400) return 'relay-type-tag--error'
  return ''
}
</script>

<style scoped>
.dashboard-page {
  min-height: 100vh;
}

.stable-action-row {
  @apply flex flex-wrap items-center gap-2;
}

.stable-action-row > * {
  @apply whitespace-nowrap;
}

.runtime-status-card {
  min-height: 240px;
  height: 100%;
}

.runtime-status-list {
  display: grid;
  gap: 0.75rem;
  margin-top: 1rem;
  color: var(--app-text-muted);
  font-size: 0.875rem;
}

.runtime-status-row {
  display: grid;
  grid-template-columns: minmax(7rem, 1fr) minmax(0, auto);
  align-items: center;
  column-gap: 1rem;
  min-height: 1.5rem;
}

.runtime-status-value {
  min-width: 0;
  color: var(--app-text);
  font-variant-numeric: tabular-nums;
  overflow-wrap: anywhere;
  text-align: right;
}

.dashboard-hero {
  border-radius: 1rem;
  border: 1px solid var(--app-border-soft);
  background:
    linear-gradient(135deg, var(--app-accent-tint), transparent 58%),
    var(--app-surface-muted);
  padding: 1.25rem;
  box-shadow: var(--app-shadow-lg);
}

.dashboard-hero__badge {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  border-radius: 9999px;
  border: 1px solid color-mix(in srgb, var(--app-accent) 25%, transparent);
  background: var(--app-accent-tint);
  padding: 0.25rem 0.75rem;
  font-size: 0.75rem;
  color: var(--app-accent);
}

.dashboard-hero__title {
  margin-top: 0.75rem;
  font-size: 1.875rem;
  font-weight: 600;
  color: var(--app-text);
}

.dashboard-hero__desc {
  margin-top: 0.5rem;
  font-size: 0.875rem;
  line-height: 1.5rem;
  color: var(--app-text-secondary);
}

.dashboard-stat-card__label,
.relay-stat-card__label {
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--app-text-muted);
}

.dashboard-stat-card__value,
.relay-stat-card__value {
  margin-top: 0.5rem;
  font-size: 1.5rem;
  font-weight: 600;
  color: var(--app-text);
}

.relay-stat-card__value--sm {
  font-size: 0.875rem;
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-stat-card__sub,
.relay-stat-card__sub {
  margin-top: 0.25rem;
  font-size: 0.875rem;
  color: var(--app-text-muted);
}

.dashboard-mini-stat {
  min-width: 0;
  border-radius: 0.75rem;
  border: 1px solid var(--app-border-soft);
  background: color-mix(in srgb, var(--app-surface) 82%, transparent);
  padding: 1rem;
}

.dashboard-mini-stat--highlight {
  border-color: color-mix(in srgb, var(--app-accent) 32%, transparent);
  background: color-mix(in srgb, var(--app-accent-tint) 58%, var(--app-surface));
}

.dashboard-mini-stat--highlight .relay-stat-card__label {
  color: var(--app-accent);
}

.relay-panel {
  border-radius: 1rem;
  border: 1px solid var(--app-border-soft);
  background:
    linear-gradient(135deg, color-mix(in srgb, var(--app-accent-tint) 85%, transparent), transparent 62%),
    var(--app-surface-muted);
  padding: 1.25rem;
  box-shadow: var(--app-shadow-lg);
}

.relay-panel__badge {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  border-radius: 9999px;
  border: 1px solid color-mix(in srgb, var(--app-accent) 25%, transparent);
  background: var(--app-accent-tint);
  padding: 0.25rem 0.75rem;
  font-size: 0.75rem;
  color: var(--app-accent);
}

.relay-panel__title {
  margin-top: 0.75rem;
  font-size: 1.125rem;
  font-weight: 600;
  color: var(--app-text);
}

.relay-panel__subtitle {
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--app-text);
}

.relay-panel__desc {
  margin-top: 0.25rem;
  font-size: 0.875rem;
  color: var(--app-text-muted);
}

.relay-panel__status-ok {
  color: var(--app-success);
}

.relay-panel__status-warn {
  color: var(--app-warning);
}

.relay-panel__hint {
  font-size: 0.75rem;
  color: var(--app-text-muted);
}

.relay-call-empty,
.relay-call-wrap {
  border-radius: 0.75rem;
  border: 1px solid var(--app-border);
  background: var(--app-surface);
}

.relay-call-empty {
  padding: 2rem 1rem;
  text-align: center;
  font-size: 0.875rem;
  color: var(--app-text-muted);
}

.relay-type-tag {
  display: inline-block;
  border-radius: 0.25rem;
  background: var(--app-control-bg);
  padding: 0.125rem 0.5rem;
  font-size: 0.75rem;
  color: var(--app-text-secondary);
}

.relay-type-tag--ok {
  background: color-mix(in srgb, var(--app-success) 14%, var(--app-control-bg));
  color: var(--app-success);
}

.relay-type-tag--error {
  background: color-mix(in srgb, var(--app-danger) 14%, var(--app-control-bg));
  color: var(--app-danger);
}

.relay-type-tag--codex {
  background: color-mix(in srgb, var(--app-accent) 14%, var(--app-control-bg));
  color: var(--app-accent);
}

.relay-type-tag--relay,
.relay-type-tag--token {
  background: color-mix(in srgb, var(--app-success) 14%, var(--app-control-bg));
  color: var(--app-success);
}

.relay-call-table .cell-muted {
  color: var(--app-text-muted);
}

.relay-call-table .cell-accent {
  color: var(--app-accent);
}

.relay-call-table .cell-text {
  color: var(--app-text);
}

.relay-call-table .cell-secondary {
  color: var(--app-text-secondary);
}

.relay-call-table thead th {
  padding: 0.75rem 1rem;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--app-text-muted);
  background: var(--app-surface-muted);
  border-bottom: 1px solid var(--app-border);
}

.relay-call-table tbody td {
  padding: 0.75rem 1rem;
  border-bottom: 1px solid color-mix(in srgb, var(--app-border) 60%, transparent);
}

.relay-call-table tbody tr:last-child td {
  border-bottom: none;
}

.relay-call-table tbody tr:hover {
  background: color-mix(in srgb, var(--app-control-bg) 70%, transparent);
}

@media (max-width: 720px) {
  .dashboard-page {
    gap: 0.875rem;
    padding: 0.875rem;
  }

  .dashboard-hero,
  .relay-panel {
    padding: 1rem;
  }

  .dashboard-hero__title {
    font-size: 1.5rem;
  }

  .dashboard-hero__desc {
    font-size: 0.8125rem;
    line-height: 1.35rem;
  }
}
</style>
