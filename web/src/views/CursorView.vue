<template>
  <div class="p-3.5 sm:p-5 lg:p-6 space-y-6">
    <!-- Header -->
    <div class="flex flex-col gap-3.5 lg:flex-row lg:items-center lg:justify-between">
      <div class="flex items-center gap-3 shrink-0">
        <span class="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl bg-gradient-to-br from-purple-500/20 to-indigo-600/30 border border-purple-500/30 text-2xl shadow-lg shadow-indigo-950/40">
          ⚡
        </span>
        <div class="min-w-0">
          <h1 class="text-xl font-bold text-white tracking-tight">Cursor 渠道管理</h1>
          <p class="text-gray-400 text-xs mt-0.5">Cursor 账号凭据与 Fast 额度监控</p>
        </div>
      </div>
      <div class="stable-actions flex flex-wrap items-center gap-2 shrink-0">
        <button
          @click="detectLocal"
          :disabled="detectingLocal"
          class="btn btn-secondary header-action-btn border-purple-500/30 hover:border-purple-500/60 text-purple-200"
          title="自动扫描本地 Cursor IDE (state.vscdb) 的登录状态并导入"
        >
          <svg class="w-4 h-4 text-purple-400" :class="{ 'animate-spin': detectingLocal }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"/>
          </svg>
          {{ detectingLocal ? '检测本地中...' : '检测本地' }}
        </button>
        <button @click="showImportModal = true" class="btn btn-secondary header-action-btn" title="批量导入账号">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"/>
          </svg>
          导入
        </button>
        <button @click="openAddModal" class="btn btn-primary header-action-btn" title="手动添加账号">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
          </svg>
          添加账号
        </button>
        <button @click="refreshAll" :disabled="refreshingAll" class="btn btn-secondary header-action-btn" title="全局刷新所有账号配额">
          <svg class="w-4 h-4" :class="{ 'animate-spin': refreshingAll }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
          </svg>
          刷新配额
        </button>
        <button @click="exportAccounts" class="btn btn-secondary header-action-btn" title="导出账号配置">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
          </svg>
          导出
        </button>
        <button
          @click="togglePrivacyMode"
          class="btn header-action-btn"
          :class="privacyMode ? 'bg-purple-500/20 text-purple-300 border-purple-500/40' : 'btn-secondary'"
          :title="privacyMode ? '隐私模式已开启：点击显示账号与头像' : '点击开启隐私模式：隐藏账号ID、邮箱、名称与头像'"
        >
          <svg v-if="privacyMode" class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21"/>
          </svg>
          <svg v-else class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"/>
          </svg>
          {{ privacyMode ? '隐私已开启' : '隐私模式' }}
        </button>
      </div>
    </div>

    <!-- Toolbar / Filter -->
    <div class="card p-4">
      <div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <div class="flex flex-1 flex-wrap items-center gap-2.5">
          <div class="relative flex-1 min-w-[200px] max-w-sm">
            <input
              v-model="searchQuery"
              type="text"
              class="input w-full pl-9"
              placeholder="搜索邮箱、名称或备注..."
            />
            <svg class="w-4 h-4 text-gray-500 absolute left-3 top-2.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
            </svg>
          </div>
          <select v-model="tierFilter" class="input w-36">
            <option value="all">所有会员等级</option>
            <option value="pro">Pro 会员</option>
            <option value="business">Business / Team</option>
            <option value="free">Free 免费版</option>
            <option value="error">异常 / 失效</option>
          </select>
          <select v-model="sortOrder" class="input w-36">
            <option value="default">默认排序</option>
            <option value="quota_high">Fast 剩余从多到少</option>
            <option value="quota_low">Fast 剩余从少到多</option>
            <option value="ondemand_high">按量消耗从多到少</option>
            <option value="email">按邮箱排序</option>
          </select>
        </div>
        <div class="flex items-center gap-3 self-end md:self-center">
          <div class="flex items-center gap-1.5 text-xs text-gray-400">
            <span>自动刷新:</span>
            <select v-model="autoRefreshMinutes" @change="setupAutoRefresh" class="input input-sm py-1 px-2 text-xs w-auto">
              <option :value="0">关闭</option>
              <option :value="1">1 分钟</option>
              <option :value="5">5 分钟</option>
              <option :value="10">10 分钟</option>
            </select>
          </div>
          <!-- 视图模式切换 -->
          <div class="flex items-center rounded-xl border border-purple-500/30 bg-purple-500/10 p-0.5 shrink-0">
            <button
              @click="setViewMode('grid')"
              class="px-2.5 py-1 text-xs font-medium rounded-lg transition-colors flex items-center gap-1.5"
              :class="viewMode === 'grid' ? 'bg-purple-600 text-white font-bold shadow-sm' : 'text-purple-200/80 hover:text-white'"
              title="卡片网格模式"
            >
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z"/>
              </svg>
              <span>卡片</span>
            </button>
            <button
              @click="setViewMode('compact')"
              class="px-2.5 py-1 text-xs font-medium rounded-lg transition-colors flex items-center gap-1.5"
              :class="viewMode === 'compact' ? 'bg-purple-600 text-white font-bold shadow-sm' : 'text-purple-200/80 hover:text-white'"
              title="紧凑列表模式"
            >
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"/>
              </svg>
              <span>紧凑</span>
            </button>
          </div>
          <span class="text-xs text-gray-400 font-medium whitespace-nowrap">共 {{ filteredAccounts.length }} 个账号</span>
          <button
            v-if="selectedIds.length > 0"
            @click="deleteSelected"
            class="btn btn-sm btn-danger ml-2"
          >
            批量删除 ({{ selectedIds.length }})
          </button>
        </div>
      </div>
    </div>

    <!-- Account List / Grid -->
    <div v-if="loading" class="text-center py-16 text-gray-400">
      <svg class="w-8 h-8 animate-spin mx-auto text-purple-500 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
      </svg>
      加载 Cursor 账号中...
    </div>

    <div v-else-if="filteredAccounts.length === 0" class="card py-16 text-center text-gray-500 space-y-3">
      <div class="text-4xl">⚡</div>
      <div class="text-base text-gray-300">暂无符合条件的 Cursor 账号</div>
      <p class="text-xs text-gray-500 max-w-md mx-auto">
        若本机已安装并登录 Cursor，可直接点击下方「检测本地 Cursor」快速导入；亦可点击「添加账号」手动粘贴 Session Token。
      </p>
      <div class="pt-2 flex justify-center gap-3">
        <button @click="detectLocal" :disabled="detectingLocal" class="btn btn-primary btn-sm">
          {{ detectingLocal ? '检测中...' : '一键检测本地 Cursor' }}
        </button>
        <button @click="openAddModal" class="btn btn-secondary btn-sm">手动添加账号</button>
      </div>
    </div>

    <!-- 紧凑列表模式 -->
    <div v-else-if="viewMode === 'compact'" class="rounded-2xl border border-purple-500/30 bg-slate-900/90 shadow-2xl backdrop-blur-xl overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-left text-xs">
          <thead class="border-b border-purple-500/25 bg-purple-500/15 text-purple-200 font-semibold tracking-wider text-[11px] uppercase">
            <tr>
              <th class="p-3.5 w-10">
                <input
                  type="checkbox"
                  :checked="selectedIds.length === filteredAccounts.length && filteredAccounts.length > 0"
                  @change="toggleSelectAll"
                  class="h-4 w-4 rounded border-purple-500/40 bg-slate-800 text-purple-500 focus:ring-0"
                />
              </th>
              <th class="p-3.5 min-w-[210px]">账号身份</th>
              <th class="p-3.5 min-w-[170px]">Fast 请求 (Included)</th>
              <th class="p-3.5 min-w-[120px]">超额按量</th>
              <th class="p-3.5 min-w-[120px]">Auto / API</th>
              <th class="p-3.5 min-w-[130px]">账单重置</th>
              <th class="p-3.5 w-24">状态</th>
              <th class="p-3.5 text-right min-w-[210px]">操作</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-800/80 text-slate-200">
            <tr
              v-for="acc in filteredAccounts"
              :key="acc.id"
              class="transition-colors duration-150"
              :class="acc.active ? 'bg-purple-500/15 hover:bg-purple-500/20 border-l-4 border-purple-400 font-medium' : 'bg-transparent hover:bg-slate-800/50'"
            >
              <td class="p-3.5">
                <input
                  type="checkbox"
                  :checked="selectedIds.includes(acc.id)"
                  @change="toggleSelect(acc.id)"
                  class="h-4 w-4 rounded border-slate-600 bg-slate-800 text-purple-500 focus:ring-0"
                />
              </td>

              <td class="p-3.5">
                <div class="flex items-center gap-2.5">
                  <div class="h-8 w-8 rounded-lg bg-gradient-to-br from-purple-600 to-indigo-700 flex items-center justify-center text-white font-bold text-xs shrink-0 overflow-hidden shadow-sm border border-purple-400/40">
                    <template v-if="privacyMode">🔒</template>
                    <template v-else>
                      <span>⚡</span>
                    </template>
                  </div>
                  <div class="min-w-0">
                    <div class="flex items-center gap-1.5">
                      <span class="font-semibold text-white truncate max-w-[160px]" :title="privacyMode ? '已隐藏' : acc.email">
                        {{ getAccountDisplayName(acc) }}
                      </span>
                      <span class="px-1.5 py-0.2 text-[9px] font-mono rounded uppercase shrink-0 border" :class="getTierBadgeClass(acc.membership_type)">
                        {{ acc.membership_type || 'FREE' }}
                      </span>
                      <span v-if="acc.active" class="px-1.5 py-0.2 text-[9px] rounded-full bg-emerald-500/25 text-emerald-300 border border-emerald-500/40 shrink-0 font-semibold flex items-center gap-1">
                        <span class="h-1 w-1 rounded-full bg-emerald-400"></span>当前生效
                      </span>
                      <span v-if="acc.tag_name" class="px-1.5 py-0.2 text-[9px] rounded text-white shrink-0 shadow-sm" :style="{ backgroundColor: acc.tag_color || '#6B7280' }">
                        {{ acc.tag_name }}
                      </span>
                    </div>
                    <div v-if="!privacyMode && acc.email && acc.display_name && acc.display_name !== acc.email" class="text-[11px] text-slate-400 truncate max-w-[160px]">
                      {{ acc.email }}
                    </div>
                  </div>
                </div>
              </td>

              <!-- Fast Requests -->
              <td class="p-3.5">
                <div class="space-y-1.5 max-w-[160px]">
                  <div class="flex items-center justify-between text-[11px]">
                    <span class="font-mono text-slate-300">
                      {{ acc.plan_used }} / {{ acc.plan_limit }}
                    </span>
                    <span :class="getQuotaColorClass(acc.plan_percentage)" class="font-mono font-bold">
                      {{ acc.plan_percentage }}%
                    </span>
                  </div>
                  <div class="h-2 w-full rounded-full bg-slate-800 border border-slate-700/60 overflow-hidden p-0.5">
                    <div
                      class="h-full rounded-full transition-all duration-300"
                      :class="getQuotaBarClass(acc.plan_percentage)"
                      :style="{ width: `${Math.min(100, Math.max(0, acc.plan_percentage))}%` }"
                    ></div>
                  </div>
                  <div class="text-[10px] text-slate-400">剩余 {{ acc.plan_remaining }} 次快速请求</div>
                </div>
              </td>

              <!-- On-Demand -->
              <td class="p-3.5 font-mono">
                <div class="font-bold text-indigo-300 text-xs">${{ (acc.on_demand_used_cents / 100).toFixed(2) }}</div>
                <div class="text-[10px] text-slate-400">
                  {{ acc.on_demand_limit_cents ? `上限 $${(acc.on_demand_limit_cents / 100).toFixed(2)}` : '按量后付费' }}
                </div>
              </td>

              <!-- Auto / API -->
              <td class="p-3.5 text-[11px] font-mono">
                <div class="text-slate-300">Auto: <b class="text-white">{{ acc.auto_percent_used }}%</b></div>
                <div class="text-slate-400">API: <b class="text-slate-300">{{ acc.api_percent_used }}%</b></div>
              </td>

              <!-- Billing reset -->
              <td class="p-3.5 text-[11px] text-slate-300">
                <div>{{ formatBillingCycle(acc.billing_cycle_end) }}</div>
                <div v-if="acc.last_refresh_at" class="text-[10px] text-slate-400">{{ formatTime(acc.last_refresh_at) }}</div>
              </td>

              <!-- Status -->
              <td class="p-3.5">
                <span v-if="acc.status === 'active' || !acc.status" class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[11px] font-semibold bg-emerald-500/15 text-emerald-300 border border-emerald-500/30">
                  <span class="h-1.5 w-1.5 rounded-full bg-emerald-400"></span> 正常
                </span>
                <span v-else-if="acc.status === 'forbidden'" class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[11px] font-semibold bg-rose-500/15 text-rose-300 border border-rose-500/30" title="403 Forbidden">
                  <span class="h-1.5 w-1.5 rounded-full bg-rose-400"></span> 受限
                </span>
                <span v-else-if="acc.status === 'expired'" class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[11px] font-semibold bg-amber-500/15 text-amber-300 border border-amber-500/30" title="Token 已失效">
                  <span class="h-1.5 w-1.5 rounded-full bg-amber-400"></span> 过期
                </span>
                <span v-else class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[11px] font-medium bg-slate-800 text-slate-300 border border-slate-700">
                  {{ acc.status }}
                </span>
              </td>

              <!-- Actions -->
              <td class="p-3.5 text-right">
                <div class="flex items-center justify-end gap-1.5">
                  <button
                    @click="activateAccount(acc)"
                    :disabled="acc.active || activatingId === acc.id"
                    class="btn btn-xs"
                    :class="acc.active ? 'border border-emerald-500/30 bg-emerald-500/10 text-emerald-300 font-semibold cursor-default' : 'btn-primary'"
                    :title="acc.active ? '当前生效中' : '设为当前生效账号并同步写入本机 Cursor IDE'"
                  >
                    <span v-if="activatingId === acc.id" class="animate-spin text-[10px]">⏳</span>
                    <span v-else-if="acc.active">✓ 生效中</span>
                    <span v-else>设为当前</span>
                  </button>
                  <button
                    @click="refreshAccount(acc)"
                    :disabled="refreshingId === acc.id"
                    class="btn btn-xs btn-secondary"
                    title="刷新配额"
                  >
                    <span v-if="refreshingId === acc.id" class="animate-spin text-[10px]">⏳</span>
                    <svg v-else class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
                    </svg>
                  </button>
                  <button
                    @click="openWakeupModal(acc)"
                    class="btn btn-xs btn-secondary"
                    title="连通性测试"
                  >
                    ⚡
                  </button>
                  <button
                    @click="copyToken(acc)"
                    class="btn btn-xs btn-secondary"
                    title="复制 Token"
                  >
                    <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/>
                    </svg>
                  </button>
                  <button
                    @click="openEditModal(acc)"
                    class="btn btn-xs btn-secondary"
                    title="编辑"
                  >
                    <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/>
                    </svg>
                  </button>
                  <button
                    @click="deleteAccount(acc)"
                    class="btn btn-xs text-rose-400 hover:text-rose-300 hover:bg-rose-500/10 p-1 rounded"
                    title="删除"
                  >
                    <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                    </svg>
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- 卡片网格模式 -->
    <div v-else class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
      <div
        v-for="acc in filteredAccounts"
        :key="acc.id"
        class="rounded-2xl border border-slate-700/70 bg-slate-900/90 p-5 space-y-4 transition-all duration-200 shadow-xl relative overflow-hidden backdrop-blur-xl hover:border-purple-500/40"
        :class="{ 'ring-2 ring-purple-500/60 border-purple-500/50 bg-gradient-to-b from-purple-950/25 to-slate-900/90': acc.active }"
      >
        <!-- Card Header -->
        <div class="flex items-center justify-between gap-3">
          <div class="flex items-center gap-3 min-w-0">
            <input
              type="checkbox"
              :checked="selectedIds.includes(acc.id)"
              @change="toggleSelect(acc.id)"
              class="h-4 w-4 rounded border-gray-700 bg-gray-900 text-purple-500 focus:ring-0 shrink-0"
            />
            <div class="relative h-10 w-10 shrink-0 rounded-xl bg-gradient-to-br from-purple-600 to-indigo-800 flex items-center justify-center text-white font-semibold text-sm shadow overflow-hidden border border-purple-500/30">
              <template v-if="privacyMode">
                <svg class="w-5 h-5 text-purple-200" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"/>
                </svg>
              </template>
              <template v-else>
                <img v-if="acc.picture" :src="acc.picture" alt="" class="h-full w-full object-cover" />
                <span v-else>{{ (acc.display_name || acc.email || 'C').slice(0, 1).toUpperCase() }}</span>
              </template>
            </div>
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-2">
                <span class="font-semibold text-white text-sm truncate" :title="privacyMode ? '已隐藏' : acc.email">
                  {{ getAccountDisplayName(acc) }}
                </span>
                <span
                  class="px-2 py-0.5 text-[10px] font-semibold rounded-md border shrink-0 tracking-wide uppercase font-mono"
                  :class="getTierBadgeClass(acc.membership_type)"
                >
                  {{ acc.membership_type || 'FREE' }}
                </span>
                <span v-if="acc.active" class="px-1.5 py-0.5 text-[10px] font-semibold rounded-md bg-emerald-500/20 text-emerald-400 border border-emerald-500/30 shrink-0">
                  当前生效
                </span>
              </div>
              <div v-if="!privacyMode && acc.email && acc.display_name && acc.display_name !== acc.email" class="text-xs text-gray-400 truncate mt-0.5" :title="acc.email">
                {{ acc.email }}
              </div>
            </div>
          </div>

          <div class="flex items-center gap-1.5 shrink-0">
            <span
              v-if="privacyMode"
              class="px-2 py-0.5 text-[10px] font-medium rounded-full bg-purple-500/10 text-purple-300 border border-purple-500/20 flex items-center gap-1"
              title="已隐藏账号详情"
            >
              🔒 隐私
            </span>
            <span
              v-if="acc.tag_name"
              class="px-2 py-0.5 text-[10px] font-medium rounded text-white"
              :style="{ backgroundColor: acc.tag_color || '#6B7280' }"
            >
              {{ acc.tag_name }}
            </span>
          </div>
        </div>

        <!-- Status warning if any (基本报错一致: 统一大小、统一颜色) -->
        <div v-if="acc.status === 'forbidden'" class="rounded-lg bg-rose-500/10 border border-rose-500/20 p-2 text-xs text-rose-400 flex items-center gap-1.5">
          <span class="shrink-0">⚠️</span>
          <span>403 Forbidden: 账号无访问权限或已被风控</span>
        </div>
        <div v-else-if="acc.status === 'expired'" class="rounded-lg bg-amber-500/10 border border-amber-500/20 p-2 text-xs text-amber-400 flex items-center gap-1.5">
          <span class="shrink-0">⚠️</span>
          <span>Token 已过期或失效，请点击刷新或重新配置</span>
        </div>
        <div v-else-if="acc.status === 'error'" class="rounded-lg bg-rose-500/10 border border-rose-500/20 p-2 text-xs text-rose-400 flex items-center gap-1.5">
          <span class="shrink-0">⚠️</span>
          <span>配额拉取异常: {{ acc.status_message || '未知错误' }}</span>
        </div>

        <!-- Quota Monitoring Section -->
        <div class="space-y-3 pt-2 border-t border-slate-800/80">
          <div class="flex items-center justify-between text-xs text-gray-400">
            <span class="font-medium">配额使用监控</span>
            <span v-if="acc.last_refresh_at" class="text-[11px] text-gray-400">
              更新于 {{ formatTime(acc.last_refresh_at) }}
            </span>
          </div>

          <!-- Included Fast Requests Bucket -->
          <div class="rounded-xl bg-slate-800/80 border border-slate-700/70 p-3.5 space-y-2 hover:border-purple-500/40 transition-colors shadow-inner">
            <div class="flex items-center justify-between">
              <span class="text-gray-200 text-xs font-semibold">Fast 请求 (Included)</span>
              <div class="flex items-center gap-2">
                <span class="font-mono text-xs text-gray-400">
                  {{ acc.plan_used }} / {{ acc.plan_limit }}
                </span>
                <span :class="getQuotaColorClass(acc.plan_percentage)" class="font-mono text-xs font-bold">
                  {{ acc.plan_percentage }}%
                </span>
              </div>
            </div>

            <!-- Progress bar (Remaining %) -->
            <div class="h-2.5 w-full rounded-full bg-slate-900 border border-slate-700/60 p-0.5 overflow-hidden">
              <div
                class="h-full rounded-full transition-all duration-300"
                :class="getQuotaBarClass(acc.plan_percentage)"
                :style="{ width: `${Math.min(100, Math.max(0, acc.plan_percentage))}%` }"
              ></div>
            </div>

            <div class="flex items-center justify-between text-[10px] text-gray-400 pt-0.5">
              <span>剩余 {{ acc.plan_remaining }} 次快速请求</span>
              <span v-if="acc.billing_cycle_end" class="flex items-center gap-1">
                <svg class="w-3 h-3 text-gray-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/>
                </svg>
                <span>账单重置: {{ formatBillingCycle(acc.billing_cycle_end) }}</span>
              </span>
            </div>
          </div>

          <!-- Secondary Metrics Grid: On-Demand & Sub-metrics -->
          <div class="grid grid-cols-2 gap-2.5">
            <!-- On-Demand Spending -->
            <div class="rounded-xl bg-slate-800/70 border border-slate-700/60 p-2.5 space-y-1 hover:border-purple-500/30 transition-colors shadow-inner">
              <div class="text-[11px] text-gray-300 flex items-center justify-between">
                <span>超额按量 (On-Demand)</span>
              </div>
              <div class="text-sm font-bold font-mono text-purple-300">
                ${{ (acc.on_demand_used_cents / 100).toFixed(2) }}
              </div>
              <div class="text-[10px] text-gray-400 truncate">
                {{ acc.on_demand_limit_cents ? `上限: $${(acc.on_demand_limit_cents / 100).toFixed(2)}` : '按量后付费消耗' }}
              </div>
            </div>

            <!-- Auto & API Usage -->
            <div class="rounded-xl bg-slate-800/70 border border-slate-700/60 p-2.5 space-y-1 hover:border-purple-500/30 transition-colors shadow-inner">
              <div class="text-[11px] text-gray-300 flex items-center justify-between">
                <span>Auto / Composer</span>
                <span class="text-purple-200 font-mono text-[11px] font-semibold">{{ acc.auto_percent_used }}%</span>
              </div>
              <div class="text-[11px] text-gray-300 flex items-center justify-between pt-1">
                <span>API Usage</span>
                <span class="text-purple-200 font-mono text-[11px] font-semibold">{{ acc.api_percent_used }}%</span>
              </div>
              <div class="text-[10px] text-gray-400 truncate pt-0.5">
                状态: {{ acc.subscription_status || 'active' }}
              </div>
            </div>
          </div>
        </div>

        <!-- Notes if any -->
        <div v-if="acc.notes" class="text-xs text-gray-300 italic bg-slate-800/60 p-2.5 rounded-lg border border-slate-700/60 truncate">
          {{ acc.notes }}
        </div>

        <!-- Card Actions -->
        <div class="flex items-center justify-between pt-3 border-t border-slate-800/80 gap-2">
          <!-- 主操作：设为当前 -->
          <button
            @click="activateAccount(acc)"
            :disabled="acc.active || activatingId === acc.id"
            class="btn btn-sm"
            :class="acc.active ? 'border border-emerald-500/30 bg-emerald-500/10 text-emerald-300 font-semibold cursor-default' : 'btn-primary'"
            :title="acc.active ? '当前生效中，已同步到 Cursor IDE' : '设为当前生效账号并同步写入本机 Cursor IDE 配置'"
          >
            <span v-if="activatingId === acc.id" class="animate-spin text-xs">⏳</span>
            <span v-else-if="acc.active" class="text-xs">✓</span>
            <span v-else class="text-xs">⚡</span>
            <span>{{ acc.active ? '当前生效' : '设为当前' }}</span>
          </button>

          <!-- 次级操作按钮组：标准高度 32px 图标按钮 -->
          <div class="flex items-center gap-1.5">
            <button
              @click="refreshAccount(acc)"
              :disabled="refreshingId === acc.id"
              class="btn btn-sm btn-secondary !px-2.5"
              title="刷新配额"
            >
              <svg class="w-3.5 h-3.5" :class="{ 'animate-spin': refreshingId === acc.id }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
              </svg>
            </button>
            <button
              @click="openWakeupModal(acc)"
              class="btn btn-sm btn-secondary !px-2.5"
              title="唤醒与连通性测试"
            >
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"/>
              </svg>
            </button>
            <button
              @click="copyToken(acc)"
              class="btn btn-sm btn-secondary !px-2.5"
              title="复制 Session Token / Access Token"
            >
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 5H6a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2v-1M8 5a2 2 0 002 2h2a2 2 0 002-2M8 5a2 2 0 012-2h2a2 2 0 012 2m0 0h2a2 2 0 012 2v3m2 4H10m0 0l3-3m-3 3l3 3"/>
              </svg>
            </button>
            <button
              @click="openEditModal(acc)"
              class="btn btn-sm btn-secondary !px-2.5"
              title="编辑账号"
            >
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z"/>
              </svg>
            </button>
            <button
              @click="deleteAccount(acc)"
              class="btn btn-sm btn-secondary hover:!bg-red-500/20 hover:!border-red-500/40 hover:!text-red-300 !px-2.5"
              title="删除账号"
            >
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
              </svg>
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- ── Add / Edit Modal ────────────────────────────────────────── -->
    <div v-if="accountModal.open" class="modal-overlay" @click.self="accountModal.open = false">
      <div class="modal-content max-w-md space-y-4">
        <div class="modal-header">
          <h3 class="font-semibold text-white">{{ accountModal.isEdit ? '编辑 Cursor 账号' : '添加 Cursor 账号' }}</h3>
          <button @click="accountModal.open = false" class="text-gray-400 hover:text-white">✕</button>
        </div>

        <div class="modal-body space-y-3">
          <div>
            <label class="label">Session Token 或 Access Token *</label>
            <textarea
              v-model="accountModal.form.session_token"
              class="input h-28 font-mono text-xs"
              placeholder="可直接粘贴 Cursor Access Token (JWT)、WorkosCursorSessionToken Cookie，或 user_xxx::ey... 格式"
            ></textarea>
            <p class="text-[11px] text-gray-500 mt-1">
              支持直接从浏览器 Cookie 或 Cursor 本地数据库 state.vscdb 获取的 Token。
            </p>
          </div>

          <div>
            <label class="label">邮箱 (可选，留空将自动从 Token 中解析)</label>
            <input
              v-model="accountModal.form.email"
              type="email"
              class="input"
              placeholder="user@example.com"
            />
          </div>

          <div>
            <label class="label">显示名称 (可选)</label>
            <input v-model="accountModal.form.display_name" type="text" class="input" placeholder="如：主力 Pro 账号" />
          </div>

          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="label">标签名</label>
              <input v-model="accountModal.form.tag_name" type="text" class="input" placeholder="如：团队/个人" />
            </div>
            <div>
              <label class="label">标签颜色</label>
              <input v-model="accountModal.form.tag_color" type="color" class="input h-10 p-1" />
            </div>
          </div>

          <div>
            <label class="label">备注 (可选)</label>
            <textarea v-model="accountModal.form.notes" class="input h-16 text-xs" placeholder="记录账号购买时间、账单日等"></textarea>
          </div>
        </div>

        <div class="modal-footer">
          <button @click="accountModal.open = false" class="btn btn-secondary">取消</button>
          <button @click="saveAccount" class="btn btn-primary" :disabled="savingAccount">
            {{ savingAccount ? '保存中...' : '保存并查询配额' }}
          </button>
        </div>
      </div>
    </div>

    <!-- ── Import Modal ────────────────────────────────────────────── -->
    <div v-if="showImportModal" class="modal-overlay" @click.self="showImportModal = false">
      <div class="modal-content max-w-lg space-y-4">
        <div class="modal-header">
          <h3 class="font-semibold text-white">批量导入 Cursor 账号</h3>
          <button @click="showImportModal = false" class="text-gray-400 hover:text-white">✕</button>
        </div>

        <div class="modal-body space-y-3">
          <div class="flex gap-2 border-b border-gray-800 pb-2">
            <button
              @click="importMode = 'tokens'"
              class="btn btn-xs"
              :class="importMode === 'tokens' ? 'btn-primary' : 'btn-secondary'"
            >
              每行一个 Token
            </button>
            <button
              @click="importMode = 'json'"
              class="btn btn-xs"
              :class="importMode === 'json' ? 'btn-primary' : 'btn-secondary'"
            >
              JSON 数组导入
            </button>
          </div>

          <div v-if="importMode === 'tokens'">
            <label class="label">每行一个 Session Token 或 Access Token</label>
            <textarea
              v-model="importContent"
              class="input h-48 font-mono text-xs"
              placeholder="eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...&#10;user_01JH...::eyJhbGciOiJSUzI1NiIs..."
            ></textarea>
          </div>

          <div v-else>
            <label class="label">粘贴账号 JSON 数组</label>
            <textarea
              v-model="importContent"
              class="input h-48 font-mono text-xs"
              placeholder='[
  { "session_token": "user_01...::ey...", "email": "user1@example.com" },
  { "session_token": "eyJhbGci...", "email": "user2@example.com" }
]'
            ></textarea>
          </div>
        </div>

        <div class="modal-footer">
          <button @click="showImportModal = false" class="btn btn-secondary">取消</button>
          <button @click="submitImport" class="btn btn-primary" :disabled="importing">
            {{ importing ? '导入并查询配额中...' : '开始导入' }}
          </button>
        </div>
      </div>
    </div>

    <!-- ── Wakeup Modal ────────────────────────────────────────────── -->
    <div v-if="wakeupModal.open" class="modal-overlay" @click.self="wakeupModal.open = false">
      <div class="modal-content max-w-lg space-y-4">
        <div class="modal-header">
          <div class="flex items-center gap-2">
            <span class="text-xl">⚡</span>
            <h3 class="font-semibold text-white">Cursor 连通性与唤醒验证</h3>
          </div>
          <button @click="wakeupModal.open = false" class="text-gray-400 hover:text-white">✕</button>
        </div>

        <div class="modal-body space-y-3">
          <div class="text-xs text-gray-400">
            通过 Cursor 官方 API 接口验证账号授权合法性并测量接口响应延迟。
          </div>

          <div class="rounded-xl bg-slate-800/80 p-3 border border-slate-700/70 text-xs space-y-1">
            <div class="text-gray-400">测试账号: <span class="text-white font-mono">{{ privacyMode ? '测试账号（已隐藏）' : wakeupModal.account?.email }}</span></div>
            <div class="text-gray-400">会员方案: <span class="text-purple-400 font-mono uppercase">{{ wakeupModal.account?.membership_type || 'FREE' }}</span></div>
          </div>

          <!-- Result section -->
          <div v-if="wakeupModal.result" class="rounded-xl bg-slate-800/80 border border-slate-700/70 p-3.5 space-y-2">
            <div class="flex items-center justify-between text-xs border-b border-slate-700/70 pb-1.5">
              <span class="text-emerald-400 font-semibold">✓ 连通性测试成功</span>
              <span class="text-purple-300 font-mono">{{ wakeupModal.result.duration_ms }} ms</span>
            </div>
            <div class="grid grid-cols-2 gap-2 text-xs py-1">
              <div class="text-gray-300">会员状态: <span class="text-white font-mono">{{ wakeupModal.result.subscription_status }}</span></div>
              <div class="text-gray-300">Fast 剩余: <span class="text-purple-300 font-mono font-bold">{{ wakeupModal.result.plan_remaining }} / {{ wakeupModal.result.plan_limit }}</span></div>
            </div>
            <div class="text-xs text-gray-200 font-mono bg-slate-900/80 border border-slate-700/60 p-2.5 rounded-lg">
              {{ wakeupModal.result.message }}
            </div>
          </div>
          <div v-else-if="wakeupModal.error" class="rounded-xl bg-rose-500/10 border border-rose-500/20 p-3 text-xs text-rose-300">
            ✗ 连通性测试失败: {{ wakeupModal.error }}
          </div>
        </div>

        <div class="modal-footer">
          <button @click="wakeupModal.open = false" class="btn btn-secondary">关闭</button>
          <button @click="runWakeup" :disabled="wakeupModal.running" class="btn btn-primary">
            {{ wakeupModal.running ? '测试中...' : '立即测试' }}
          </button>
        </div>
      </div>
    </div>

  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, inject } from 'vue'
import { cursorAPI } from '@/api'

const notify = inject('notify', (msg, type) => alert(msg))

const accounts = ref([])
const loading = ref(false)
const searchQuery = ref('')
const tierFilter = ref('all')
const sortOrder = ref('default')
const selectedIds = ref([])
const activatingId = ref(null)
const refreshingId = ref(null)
const refreshingAll = ref(false)
const detectingLocal = ref(false)
const savingAccount = ref(false)
const autoRefreshMinutes = ref(0)
let autoRefreshTimer = null

const privacyMode = ref(localStorage.getItem('easyllm.cursor.privacyMode') === 'true')
const viewMode = ref(localStorage.getItem('easyllm.cursor.viewMode') || 'grid')

function setViewMode(mode) {
  viewMode.value = mode
  localStorage.setItem('easyllm.cursor.viewMode', mode)
}

function togglePrivacyMode() {
  privacyMode.value = !privacyMode.value
  localStorage.setItem('easyllm.cursor.privacyMode', privacyMode.value ? 'true' : 'false')
  notify(privacyMode.value ? '隐私模式已开启：隐藏账号ID、邮箱与头像' : '隐私模式已关闭：显示账号信息', 'info')
}

function getAccountDisplayName(acc) {
  if (privacyMode.value) {
    const idx = accounts.value.findIndex(a => a.id === acc.id)
    return `账号 #${idx >= 0 ? idx + 1 : (acc.id || '').slice(0, 4)}`
  }
  return acc.display_name || acc.email || '未命名账号'
}

function getTierBadgeClass(tier) {
  const t = (tier || '').toLowerCase()
  if (t === 'pro') return 'bg-purple-500/20 text-purple-300 border-purple-500/30'
  if (t === 'business' || t === 'enterprise') return 'bg-teal-500/20 text-teal-300 border-teal-500/30'
  return 'bg-gray-700/50 text-gray-300 border-gray-600/30'
}

function getQuotaColorClass(pct) {
  if (pct === null || pct === undefined) return 'text-gray-400'
  if (pct >= 50) return 'text-emerald-400'
  if (pct >= 20) return 'text-amber-400'
  return 'text-rose-400'
}

function getQuotaBarClass(pct) {
  if (pct === null || pct === undefined) return 'bg-gray-600'
  if (pct >= 50) return 'bg-emerald-500'
  if (pct >= 20) return 'bg-amber-500'
  return 'bg-rose-500'
}

function formatTime(isoStr) {
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
  if (!endDateStr) return '-'
  try {
    const d = new Date(endDateStr)
    const now = new Date()
    const diffDays = Math.ceil((d - now) / (1000 * 60 * 60 * 24))
    if (diffDays > 0) {
      return `${endDateStr} (${diffDays}天后)`
    }
    return endDateStr
  } catch (e) {
    return endDateStr
  }
}

const filteredAccounts = computed(() => {
  let list = accounts.value.slice()

  if (searchQuery.value.trim()) {
    const q = searchQuery.value.trim().toLowerCase()
    list = list.filter(acc =>
      (acc.email && acc.email.toLowerCase().includes(q)) ||
      (acc.display_name && acc.display_name.toLowerCase().includes(q)) ||
      (acc.notes && acc.notes.toLowerCase().includes(q))
    )
  }

  if (tierFilter.value !== 'all') {
    if (tierFilter.value === 'error') {
      list = list.filter(acc => acc.status !== 'active')
    } else {
      list = list.filter(acc => (acc.membership_type || 'free').toLowerCase() === tierFilter.value)
    }
  }

  if (sortOrder.value === 'quota_high') {
    list.sort((a, b) => (b.plan_remaining || 0) - (a.plan_remaining || 0))
  } else if (sortOrder.value === 'quota_low') {
    list.sort((a, b) => (a.plan_remaining || 0) - (b.plan_remaining || 0))
  } else if (sortOrder.value === 'ondemand_high') {
    list.sort((a, b) => (b.on_demand_used_cents || 0) - (a.on_demand_used_cents || 0))
  } else if (sortOrder.value === 'email') {
    list.sort((a, b) => (a.email || '').localeCompare(b.email || ''))
  }

  return list
})

async function fetchAccounts() {
  loading.value = true
  try {
    const res = await cursorAPI.list()
    accounts.value = res || []
  } catch (err) {
    notify('获取 Cursor 账号列表失败: ' + err.message, 'error')
  } finally {
    loading.value = false
  }
}

async function detectLocal() {
  detectingLocal.value = true
  try {
    const res = await cursorAPI.detectLocal()
    notify(res.message || '成功读取本地 Cursor 账号！', 'success')
    await fetchAccounts()
  } catch (err) {
    notify('检测本地 Cursor 账号失败: ' + err.message, 'error')
  } finally {
    detectingLocal.value = false
  }
}

async function activateAccount(acc) {
  activatingId.value = acc.id
  try {
    await cursorAPI.activate(acc.id)
    notify(`已激活账号 ${getAccountDisplayName(acc)}，并同步写回本地 Cursor IDE`, 'success')
    await fetchAccounts()
  } catch (err) {
    notify('激活失败: ' + err.message, 'error')
  } finally {
    activatingId.value = null
  }
}

async function refreshAccount(acc) {
  refreshingId.value = acc.id
  try {
    await cursorAPI.refresh(acc.id)
    notify(`账号 ${getAccountDisplayName(acc)} 配额已更新`, 'success')
    await fetchAccounts()
  } catch (err) {
    notify('刷新配额失败: ' + err.message, 'error')
  } finally {
    refreshingId.value = null
  }
}

async function refreshAll() {
  refreshingAll.value = true
  try {
    const res = await cursorAPI.refreshAll()
    notify(`全局配额刷新完成：成功 ${res.updated} 个，失败 ${res.failed} 个`, 'success')
    await fetchAccounts()
  } catch (err) {
    notify('全局刷新失败: ' + err.message, 'error')
  } finally {
    refreshingAll.value = false
  }
}

function copyToken(acc) {
  const token = acc.session_token || acc.access_token
  if (!token) {
    notify('无可用 Token', 'error')
    return
  }
  navigator.clipboard.writeText(token)
    .then(() => notify('已复制 Token 到剪贴板', 'success'))
    .catch(() => notify('复制失败，请手动复制', 'error'))
}

// ── Add / Edit Modal ─────────────────────────────────────────────
const accountModal = ref({
  open: false,
  isEdit: false,
  id: '',
  form: {
    session_token: '',
    email: '',
    display_name: '',
    tag_name: '',
    tag_color: '#8B5CF6',
    notes: '',
  }
})

function openAddModal() {
  accountModal.value = {
    open: true,
    isEdit: false,
    id: '',
    form: {
      session_token: '',
      email: '',
      display_name: '',
      tag_name: '',
      tag_color: '#8B5CF6',
      notes: '',
    }
  }
}

function openEditModal(acc) {
  accountModal.value = {
    open: true,
    isEdit: true,
    id: acc.id,
    form: {
      session_token: acc.session_token || acc.access_token || '',
      email: acc.email || '',
      display_name: acc.display_name || '',
      tag_name: acc.tag_name || '',
      tag_color: acc.tag_color || '#8B5CF6',
      notes: acc.notes || '',
    }
  }
}

async function saveAccount() {
  if (!accountModal.value.form.session_token.trim()) {
    notify('请填写 Session Token 或 Access Token', 'error')
    return
  }
  savingAccount.value = true
  try {
    if (accountModal.value.isEdit) {
      await cursorAPI.update(accountModal.value.id, accountModal.value.form)
      notify('账号已更新', 'success')
    } else {
      await cursorAPI.add(accountModal.value.form)
      notify('账号已添加并完成配额拉取', 'success')
    }
    accountModal.value.open = false
    await fetchAccounts()
  } catch (err) {
    notify('保存失败: ' + err.message, 'error')
  } finally {
    savingAccount.value = false
  }
}

async function deleteAccount(acc) {
  if (!confirm(`确定删除账号「${getAccountDisplayName(acc)}」吗？`)) return
  try {
    await cursorAPI.delete(acc.id)
    notify('账号已删除', 'success')
    await fetchAccounts()
  } catch (err) {
    notify('删除失败: ' + err.message, 'error')
  }
}

function toggleSelectAll() {
  if (selectedIds.value.length === filteredAccounts.value.length && filteredAccounts.value.length > 0) {
    selectedIds.value = []
  } else {
    selectedIds.value = filteredAccounts.value.map(a => a.id)
  }
}

function toggleSelect(id) {
  const idx = selectedIds.value.indexOf(id)
  if (idx >= 0) selectedIds.value.splice(idx, 1)
  else selectedIds.value.push(id)
}

async function deleteSelected() {
  if (!confirm(`确定删除选中的 ${selectedIds.value.length} 个账号吗？`)) return
  try {
    await cursorAPI.deleteMany(selectedIds.value)
    notify(`已删除 ${selectedIds.value.length} 个账号`, 'success')
    selectedIds.value = []
    await fetchAccounts()
  } catch (err) {
    notify('批量删除失败: ' + err.message, 'error')
  }
}

// ── Import Modal ─────────────────────────────────────────────────
const showImportModal = ref(false)
const importMode = ref('tokens')
const importContent = ref('')
const importing = ref(false)

async function submitImport() {
  const content = importContent.value.trim()
  if (!content) {
    notify('请输入导入内容', 'error')
    return
  }

  let list = []
  if (importMode.value === 'json') {
    try {
      list = JSON.parse(content)
      if (!Array.isArray(list)) throw new Error('必须为 JSON 数组')
    } catch (e) {
      notify('JSON 解析失败: ' + e.message, 'error')
      return
    }
  } else {
    list = content.split('\n')
      .map(line => line.trim())
      .filter(Boolean)
      .map(token => ({ session_token: token }))
  }

  if (list.length === 0) {
    notify('无可导入的账号数据', 'error')
    return
  }

  importing.value = true
  try {
    const res = await cursorAPI.importAccounts({ accounts: list })
    notify(`导入完成：成功 ${res.imported} 个，失败 ${res.failed} 个`, 'success')
    showImportModal.value = false
    importContent.value = ''
    await fetchAccounts()
  } catch (err) {
    notify('导入失败: ' + err.message, 'error')
  } finally {
    importing.value = false
  }
}

// ── Wakeup Modal ─────────────────────────────────────────────────
const wakeupModal = ref({
  open: false,
  running: false,
  account: null,
  result: null,
  error: null,
})

function openWakeupModal(acc) {
  wakeupModal.value = {
    open: true,
    running: false,
    account: acc,
    result: null,
    error: null,
  }
  runWakeup()
}

async function runWakeup() {
  if (!wakeupModal.value.account) return
  wakeupModal.value.running = true
  wakeupModal.value.result = null
  wakeupModal.value.error = null
  try {
    const res = await cursorAPI.wakeup(wakeupModal.value.account.id)
    wakeupModal.value.result = res
  } catch (err) {
    wakeupModal.value.error = err.message
  } finally {
    wakeupModal.value.running = false
  }
}

// ── Export ───────────────────────────────────────────────────────
async function exportAccounts() {
  try {
    const data = await cursorAPI.exportJSON()
    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `easyllm-cursor-accounts-${new Date().toISOString().slice(0, 10)}.json`
    a.click()
    URL.revokeObjectURL(url)
    notify('账号导出成功', 'success')
  } catch (err) {
    notify('导出失败: ' + err.message, 'error')
  }
}

function setupAutoRefresh() {
  if (autoRefreshTimer) {
    clearInterval(autoRefreshTimer)
    autoRefreshTimer = null
  }
  if (autoRefreshMinutes.value > 0) {
    autoRefreshTimer = setInterval(() => {
      if (document.hidden || window.__easyllm_hidden) return
      fetchAccounts()
    }, autoRefreshMinutes.value * 60 * 1000)
    notify(`已开启每 ${autoRefreshMinutes.value} 分钟自动刷新`, 'info')
  }
}

let cursorVisibilityHandler = null

onMounted(() => {
  fetchAccounts()
  cursorVisibilityHandler = () => {
    if (!document.hidden && !window.__easyllm_hidden) {
      fetchAccounts()
    }
  }
  document.addEventListener('visibilitychange', cursorVisibilityHandler)
  window.addEventListener('easyllm-app-visibility', cursorVisibilityHandler)
})

onUnmounted(() => {
  if (autoRefreshTimer) clearInterval(autoRefreshTimer)
  if (cursorVisibilityHandler) {
    document.removeEventListener('visibilitychange', cursorVisibilityHandler)
    window.removeEventListener('easyllm-app-visibility', cursorVisibilityHandler)
  }
})
</script>
