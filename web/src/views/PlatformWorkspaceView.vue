<template>
  <div class="p-6 space-y-6">
    <section
      class="rounded-3xl border border-gray-800 bg-gradient-to-br p-6 shadow-2xl shadow-black/20"
      :class="platform.heroClass"
    >
      <div class="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
        <div class="space-y-3 max-w-3xl">
          <div class="inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/5 px-3 py-1 text-xs text-gray-300">
            <PlatformIcon :platform="platform" size="xs" />
            <span>{{ platform.category }}</span>
            <span class="text-gray-500">/</span>
            <span>{{ platform.managementMode === 'legacy' ? 'legacy flow' : '通用流程' }}</span>
          </div>
          <div>
            <h1 class="text-3xl font-semibold text-white">{{ platform.label }}</h1>
            <p class="mt-2 text-sm leading-6 text-gray-300">{{ platform.description }}</p>
          </div>
          <div class="flex flex-wrap gap-2 text-xs">
            <span class="badge badge-blue">多账号</span>
            <span class="badge" :class="supportsInstances ? 'badge-green' : 'badge-gray'">实例管理</span>
            <span class="badge" :class="supportsWakeup ? 'badge-green' : 'badge-gray'">唤醒任务</span>
            <span class="badge" :class="platform.supports.quota ? 'badge-green' : 'badge-gray'">配额观察</span>
          </div>
        </div>

        <div class="grid grid-cols-2 gap-3 md:grid-cols-4">
          <div class="rounded-2xl border border-white/10 bg-black/20 p-4">
            <div class="text-xs uppercase tracking-wide text-gray-400">账号</div>
            <div class="mt-2 text-2xl font-semibold text-white">{{ accounts.length }}</div>
            <div class="mt-1 text-xs text-gray-500">已接入</div>
          </div>
          <div class="rounded-2xl border border-white/10 bg-black/20 p-4">
            <div class="text-xs uppercase tracking-wide text-gray-400">激活</div>
            <div class="mt-2 text-2xl font-semibold text-white">{{ activeAccount ? 1 : 0 }}</div>
            <div class="mt-1 truncate text-xs text-gray-500">{{ activeAccount?.email || '未指定' }}</div>
          </div>
          <div class="rounded-2xl border border-white/10 bg-black/20 p-4">
            <div class="text-xs uppercase tracking-wide text-gray-400">实例</div>
            <div class="mt-2 text-2xl font-semibold text-white">{{ instances.length }}</div>
            <div class="mt-1 text-xs text-gray-500">{{ runningInstances.length }} 运行中</div>
          </div>
          <div class="rounded-2xl border border-white/10 bg-black/20 p-4">
            <div class="text-xs uppercase tracking-wide text-gray-400">唤醒</div>
            <div class="mt-2 text-2xl font-semibold text-white">{{ wakeupTasks.length }}</div>
            <div class="mt-1 text-xs text-gray-500">{{ enabledWakeupTasks.length }} 已启用</div>
          </div>
        </div>
      </div>
    </section>

    <div class="flex flex-wrap gap-2">
      <button
        class="btn btn-sm"
        :class="activeTab === 'accounts' ? 'btn-primary' : 'btn-secondary'"
        @click="activeTab = 'accounts'"
      >
        账号台账
      </button>
      <button
        v-if="supportsInstances"
        class="btn btn-sm"
        :class="activeTab === 'instances' ? 'btn-primary' : 'btn-secondary'"
        @click="activeTab = 'instances'"
      >
        实例编排
      </button>
      <button
        v-if="supportsWakeup"
        class="btn btn-sm"
        :class="activeTab === 'wakeup' ? 'btn-primary' : 'btn-secondary'"
        @click="activeTab = 'wakeup'"
      >
        唤醒任务
      </button>
      <div class="ml-auto flex gap-2">
        <button class="btn btn-secondary btn-sm" :disabled="loading" @click="loadData">
          {{ loading ? '同步中...' : '刷新数据' }}
        </button>
      </div>
    </div>

    <section v-if="loading" class="card p-10 text-center text-gray-400">
      正在同步 {{ platform.label }} 数据...
    </section>

    <template v-else>
      <section v-if="activeTab === 'accounts'" class="card overflow-hidden">
        <div class="flex flex-col gap-4 border-b border-gray-800 px-5 py-4 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <h2 class="text-lg font-semibold text-white">账号管理</h2>
            <p class="mt-1 text-sm text-gray-500">标准化的“账号 + 配额 + 激活账号”工作流。</p>
          </div>
          <div class="flex flex-col gap-2 sm:flex-row">
            <input
              v-model="accountSearch"
              class="input w-full sm:w-64"
              placeholder="搜索邮箱 / 计划 / 标签"
            />
            <button class="btn btn-secondary" @click="exportAccounts">导出 JSON</button>
            <button class="btn btn-secondary" @click="openImportModal">导入 JSON</button>
            <button v-if="supportsOAuth" class="btn btn-secondary" @click="openOAuthModal">OAuth 授权</button>
            <button class="btn btn-primary" @click="openAccountModal()">新增账号</button>
          </div>
        </div>

        <div v-if="filteredAccounts.length === 0" class="p-10 text-center text-sm text-gray-500">
          当前平台还没有账号，先从“新增账号”开始。
        </div>
        <div v-else class="overflow-x-auto">
          <table class="min-w-full text-sm">
            <thead class="bg-gray-900/80 text-left text-gray-400">
              <tr>
                <th class="px-5 py-3">账号</th>
                <th class="px-5 py-3">计划 / 状态</th>
                <th class="px-5 py-3">配额</th>
                <th class="px-5 py-3">标签</th>
                <th class="px-5 py-3">更新时间</th>
                <th class="px-5 py-3 text-right">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="account in filteredAccounts"
                :key="account.id"
                class="border-t border-gray-800/80 bg-gray-950/40 align-top"
              >
                <td class="px-5 py-4">
                  <div class="flex items-start gap-3">
                    <div
                      class="mt-0.5 h-2.5 w-2.5 rounded-full"
                      :class="account.active ? 'bg-emerald-400' : 'bg-gray-600'"
                    />
                    <div class="min-w-0">
                      <div class="truncate font-medium text-white">{{ account.email }}</div>
                      <div class="mt-1 text-xs text-gray-500">
                        {{ account.display_name || '未填写显示名' }}
                      </div>
                      <div v-if="account.notes" class="mt-2 line-clamp-2 text-xs text-gray-400">
                        {{ account.notes }}
                      </div>
                    </div>
                  </div>
                </td>
                <td class="px-5 py-4">
                  <div class="flex flex-wrap gap-2">
                    <span class="badge" :class="account.active ? 'badge-green' : 'badge-gray'">
                      {{ account.active ? '当前激活' : '备用账号' }}
                    </span>
                    <span class="badge badge-blue">{{ account.plan || '未标注计划' }}</span>
                    <span class="badge" :class="account.status === 'disabled' ? 'badge-red' : 'badge-gray'">
                      {{ account.status || 'active' }}
                    </span>
                  </div>
                </td>
                <td class="px-5 py-4">
                  <div class="text-white">{{ quotaText(account) }}</div>
                  <div v-if="account.quota_reset_at" class="mt-1 text-xs text-gray-500">
                    重置: {{ formatDateTime(account.quota_reset_at) }}
                  </div>
                </td>
                <td class="px-5 py-4">
                  <span
                    v-if="account.tag_name"
                    class="inline-flex rounded-full px-2 py-1 text-xs font-medium text-white"
                    :style="{ backgroundColor: account.tag_color || '#4B5563' }"
                  >
                    {{ account.tag_name }}
                  </span>
                  <span v-else class="text-xs text-gray-600">未标记</span>
                </td>
                <td class="px-5 py-4 text-gray-400">
                  {{ formatDateTime(account.updated_at) }}
                </td>
                <td class="px-5 py-4">
                  <div class="flex justify-end gap-2">
                    <button class="btn btn-secondary btn-xs" @click="openAccountModal(account)">编辑</button>
                    <button class="btn btn-success btn-xs" @click="activateAccount(account)" :disabled="account.active">
                      激活
                    </button>
                    <button class="btn btn-danger btn-xs" @click="deleteAccount(account)">删除</button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <section v-else-if="activeTab === 'instances'" class="space-y-4">
        <div class="flex flex-col gap-4 rounded-2xl border border-gray-800 bg-gray-900/70 p-5 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <h2 class="text-lg font-semibold text-white">实例编排</h2>
            <p class="mt-1 text-sm text-gray-500">每个实例都可以绑定账号、工作目录和独立用户数据目录。</p>
          </div>
          <div class="flex gap-2">
            <button class="btn btn-secondary" @click="exportInstances">导出实例</button>
            <button class="btn btn-primary" @click="openInstanceModal()">新增实例</button>
          </div>
        </div>

        <div v-if="instances.length === 0" class="card p-10 text-center text-sm text-gray-500">
          还没有实例配置。你可以先创建一个绑定当前平台账号的独立实例。
        </div>
        <div v-else class="grid gap-4 lg:grid-cols-2">
          <article
            v-for="instance in instances"
            :key="instance.id"
            class="card p-5"
          >
            <div class="flex items-start justify-between gap-4">
              <div class="space-y-2">
                <div class="flex items-center gap-2">
                  <h3 class="text-base font-semibold text-white">{{ instance.name }}</h3>
                  <span class="badge" :class="instance.state === 'running' ? 'badge-green' : 'badge-gray'">
                    {{ instance.state || 'stopped' }}
                  </span>
                </div>
                <div class="text-sm text-gray-400">
                  绑定账号: {{ accountLabel(instance.account_id) }}
                </div>
              </div>
              <div class="flex gap-2">
                <button class="btn btn-secondary btn-xs" @click="setInstanceState(instance, instance.state === 'running' ? 'paused' : 'running')">
                  {{ instance.state === 'running' ? '暂停' : '启动' }}
                </button>
                <button v-if="instance.state !== 'stopped'" class="btn btn-secondary btn-xs" @click="setInstanceState(instance, 'stopped')">
                  停止
                </button>
                <button class="btn btn-secondary btn-xs" @click="openInstanceModal(instance)">编辑</button>
                <button class="btn btn-danger btn-xs" @click="deleteInstance(instance)">删除</button>
              </div>
            </div>

            <dl class="mt-4 space-y-2 text-sm text-gray-400">
              <div class="grid grid-cols-[84px_1fr] gap-3">
                <dt class="text-gray-500">工作目录</dt>
                <dd class="truncate">{{ instance.workspace_dir || '未配置' }}</dd>
              </div>
              <div class="grid grid-cols-[84px_1fr] gap-3">
                <dt class="text-gray-500">用户目录</dt>
                <dd class="truncate">{{ instance.user_data_dir || '未配置' }}</dd>
              </div>
              <div class="grid grid-cols-[84px_1fr] gap-3">
                <dt class="text-gray-500">启动参数</dt>
                <dd class="truncate">{{ instance.launch_args || '默认' }}</dd>
              </div>
              <div class="grid grid-cols-[84px_1fr] gap-3">
                <dt class="text-gray-500">最近启动</dt>
                <dd>{{ formatDateTime(instance.last_started_at) }}</dd>
              </div>
            </dl>

            <div v-if="instance.notes" class="mt-4 rounded-xl border border-gray-800 bg-gray-950/60 p-3 text-sm text-gray-300">
              {{ instance.notes }}
            </div>
          </article>
        </div>
      </section>

      <section v-else class="space-y-4">
        <div class="flex flex-col gap-4 rounded-2xl border border-gray-800 bg-gray-900/70 p-5 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <h2 class="text-lg font-semibold text-white">唤醒任务</h2>
            <p class="mt-1 text-sm text-gray-500">先保存调度台账与运行参数，后续可以继续接入真实执行器。</p>
          </div>
          <button class="btn btn-primary" @click="openWakeupModal()">新增任务</button>
        </div>

        <div v-if="wakeupTasks.length === 0" class="card p-10 text-center text-sm text-gray-500">
          当前还没有唤醒任务。
        </div>
        <div v-else class="grid gap-4 lg:grid-cols-2">
          <article v-for="task in wakeupTasks" :key="task.id" class="card p-5">
            <div class="flex items-start justify-between gap-4">
              <div class="space-y-2">
                <div class="flex items-center gap-2">
                  <h3 class="text-base font-semibold text-white">{{ task.name }}</h3>
                  <span class="badge" :class="task.enabled ? 'badge-green' : 'badge-gray'">
                    {{ task.enabled ? '启用中' : '已禁用' }}
                  </span>
                </div>
                <div class="text-sm text-gray-400">
                  账号: {{ accountLabel(task.account_id) }}
                </div>
              </div>
              <div class="flex gap-2">
                <button class="btn btn-secondary btn-xs" @click="openWakeupModal(task)">编辑</button>
                <button class="btn btn-secondary btn-xs" @click="toggleWakeup(task)">
                  {{ task.enabled ? '停用' : '启用' }}
                </button>
                <button class="btn btn-danger btn-xs" @click="deleteWakeup(task)">删除</button>
              </div>
            </div>

            <div class="mt-4 grid gap-3 text-sm text-gray-400 md:grid-cols-2">
              <div>
                <div class="text-xs uppercase tracking-wide text-gray-500">调度</div>
                <div class="mt-1 text-white">{{ scheduleText(task) }}</div>
              </div>
              <div>
                <div class="text-xs uppercase tracking-wide text-gray-500">模型</div>
                <div class="mt-1 text-white">{{ task.model || '未指定' }}</div>
              </div>
              <div>
                <div class="text-xs uppercase tracking-wide text-gray-500">最近运行</div>
                <div class="mt-1 text-white">{{ formatDateTime(task.last_run_at) }}</div>
              </div>
              <div>
                <div class="text-xs uppercase tracking-wide text-gray-500">下次计划</div>
                <div class="mt-1 text-white">{{ formatDateTime(task.next_run_at) }}</div>
              </div>
            </div>

            <div v-if="task.prompt" class="mt-4 rounded-xl border border-gray-800 bg-gray-950/60 p-3 text-sm text-gray-300">
              {{ task.prompt }}
            </div>
          </article>
        </div>
      </section>
    </template>

    <div v-if="showImportModal" class="modal-overlay" @click.self="closeImportModal">
      <div class="modal-content max-w-3xl">
        <div class="modal-header">
          <h3 class="text-white">导入 {{ platform.label }} 账号 JSON</h3>
          <button class="text-gray-500 hover:text-white" @click="closeImportModal">✕</button>
        </div>
        <div class="modal-body space-y-4">
          <div class="rounded-xl border border-amber-700/40 bg-amber-900/10 px-4 py-3 text-sm text-amber-200">
            粘贴数组 JSON 即可批量导入，字段可包含 `email`、`access_token`、`refresh_token`、`plan`、`tag_name`、`quota_used` 等。
          </div>
          <textarea
            v-model="importJson"
            class="input min-h-80 font-mono text-xs"
            placeholder='[{"email":"user@example.com","access_token":"token","plan":"pro"}]'
          />
        </div>
        <div class="modal-footer">
          <button class="btn btn-secondary mr-auto" @click="downloadImportExample">下载示例</button>
          <button class="btn btn-secondary" @click="closeImportModal">取消</button>
          <button class="btn btn-primary" @click="importAccounts">开始导入</button>
        </div>
      </div>
    </div>

    <div v-if="showAccountModal" class="modal-overlay" @click.self="closeAccountModal">
      <div class="modal-content max-w-2xl">
        <div class="modal-header">
          <h3 class="text-white">{{ editingAccount ? '编辑账号' : `新增 ${platform.label} 账号` }}</h3>
          <button class="text-gray-500 hover:text-white" @click="closeAccountModal">✕</button>
        </div>
        <div class="modal-body space-y-4">
          <div class="grid gap-4 md:grid-cols-2">
            <div>
              <label class="label">邮箱</label>
              <input v-model="accountForm.email" class="input" placeholder="name@example.com" />
            </div>
            <div>
              <label class="label">显示名</label>
              <input v-model="accountForm.display_name" class="input" placeholder="选填" />
            </div>
          </div>
          <div class="grid gap-4 md:grid-cols-2">
            <div>
              <label class="label">Access Token</label>
              <textarea v-model="accountForm.access_token" class="input min-h-28 font-mono text-xs" />
            </div>
            <div>
              <label class="label">Refresh / Cookie Token</label>
              <textarea v-model="accountForm.refresh_token" class="input min-h-28 font-mono text-xs" placeholder="可填写 refresh token，Cookie 可放到备注或 metadata" />
            </div>
          </div>
          <div class="grid gap-4 md:grid-cols-3">
            <div>
              <label class="label">计划</label>
              <input v-model="accountForm.plan" class="input" placeholder="Free / Pro / Team" />
            </div>
            <div>
              <label class="label">状态</label>
              <select v-model="accountForm.status" class="input">
                <option value="active">active</option>
                <option value="pending">pending</option>
                <option value="disabled">disabled</option>
              </select>
            </div>
            <div class="flex items-end">
              <label class="flex items-center gap-2 text-sm text-gray-300">
                <input v-model="accountForm.active" type="checkbox" class="h-4 w-4 rounded border-gray-600 bg-gray-900 text-blue-500" />
                设为当前激活
              </label>
            </div>
          </div>
          <div class="grid gap-4 md:grid-cols-4">
            <div>
              <label class="label">已用配额</label>
              <input v-model="accountForm.quota_used" class="input" placeholder="0" />
            </div>
            <div>
              <label class="label">总配额</label>
              <input v-model="accountForm.quota_limit" class="input" placeholder="100" />
            </div>
            <div>
              <label class="label">单位</label>
              <input v-model="accountForm.quota_unit" class="input" placeholder="credits" />
            </div>
            <div>
              <label class="label">标签颜色</label>
              <input v-model="accountForm.tag_color" type="color" class="input h-11 p-1" />
            </div>
          </div>
          <div class="grid gap-4 md:grid-cols-2">
            <div>
              <label class="label">标签</label>
              <input v-model="accountForm.tag_name" class="input" placeholder="重要账号 / 可切号" />
            </div>
            <div>
              <label class="label">备注</label>
              <input v-model="accountForm.notes" class="input" placeholder="补充说明" />
            </div>
          </div>
        </div>
        <div class="modal-footer">
          <button class="btn btn-secondary" @click="closeAccountModal">取消</button>
          <button class="btn btn-primary" @click="saveAccount">保存</button>
        </div>
      </div>
    </div>

    <!-- OAuth Modal -->
    <div v-if="showOAuthModal" class="modal-overlay" @click.self="closeOAuthModal">
      <div class="modal-content max-w-md">
        <div class="modal-header">
          <h3 class="text-white">{{ platform.label }} OAuth 授权</h3>
          <button class="text-gray-500 hover:text-white" @click="closeOAuthModal">✕</button>
        </div>
        <div class="modal-body space-y-4">
          <div v-if="oauthState.error" class="rounded-lg bg-red-500/10 border border-red-500/30 p-3 text-sm text-red-300">
            {{ oauthState.error }}
          </div>

          <div v-if="!oauthState.authUrl && !oauthState.loading">
            <p class="text-sm text-gray-400 mb-4">点击下方按钮开始 OAuth 授权流程</p>
            <button class="btn btn-primary w-full" @click="startOAuth">开始授权</button>
          </div>

          <div v-else-if="oauthState.loading && !oauthState.authUrl">
            <div class="flex items-center justify-center py-8">
              <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
              <span class="ml-3 text-gray-400">准备授权...</span>
            </div>
          </div>

          <div v-else-if="oauthState.authUrl" class="space-y-4">
            <div class="rounded-lg bg-blue-500/10 border border-blue-500/30 p-3 text-sm text-blue-300">
              请在浏览器中完成授权，完成后会自动添加账号。若浏览器没有自动跳回本机端口，可粘贴回调地址继续。
            </div>

            <div>
              <label class="label">授权链接</label>
              <div class="flex flex-wrap gap-2">
                <input readonly :value="oauthState.authUrl" class="input min-w-0 flex-1 text-xs font-mono" />
                <button @click="copyAuthUrl" class="btn btn-secondary text-xs px-3">复制</button>
                <button @click="openAuthUrl" class="btn btn-secondary text-xs px-3">打开</button>
              </div>
            </div>

            <!-- Device Code for GitHub Copilot -->
            <div v-if="oauthState.userCode">
              <label class="label">设备码（需要在授权页面输入）</label>
              <div class="flex gap-2">
                <input readonly :value="oauthState.userCode" class="input flex-1 text-center text-2xl font-bold tracking-widest" />
                <button @click="copyUserCode" class="btn btn-secondary text-xs px-3">复制</button>
              </div>
            </div>

            <div v-if="oauthState.manualCallback" class="space-y-2">
              <label class="label">手动回调地址</label>
              <div class="flex flex-wrap gap-2">
                <input v-model="oauthState.callbackUrl" class="input min-w-0 flex-1 text-xs font-mono" placeholder="粘贴完整回调地址或 ?code=...&state=..." />
                <button @click="submitOAuthCallback" :disabled="oauthState.callbackSubmitting || !oauthState.callbackUrl.trim()" class="btn btn-secondary text-xs px-3">
                  {{ oauthState.callbackSubmitting ? '提交中...' : '提交' }}
                </button>
              </div>
              <p v-if="oauthState.callbackHint" class="text-xs text-gray-500">{{ oauthState.callbackHint }}</p>
            </div>

            <div v-if="oauthState.waiting" class="flex items-center justify-center py-4">
              <div class="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-500"></div>
              <span class="ml-3 text-gray-400">等待授权完成...</span>
            </div>

            <div v-if="oauthState.success" class="rounded-lg bg-green-500/10 border border-green-500/30 p-3 text-sm text-green-300">
              ✓ 授权成功！账号已添加
            </div>
          </div>
        </div>
        <div class="modal-footer">
          <button class="btn btn-secondary" @click="closeOAuthModal">{{ oauthState.success ? '关闭' : '取消' }}</button>
        </div>
      </div>
    </div>

    <div v-if="showInstanceModal" class="modal-overlay" @click.self="closeInstanceModal">
      <div class="modal-content max-w-2xl">
        <div class="modal-header">
          <h3 class="text-white">{{ editingInstance ? '编辑实例' : `新增 ${platform.label} 实例` }}</h3>
          <button class="text-gray-500 hover:text-white" @click="closeInstanceModal">✕</button>
        </div>
        <div class="modal-body space-y-4">
          <div class="grid gap-4 md:grid-cols-2">
            <div>
              <label class="label">实例名称</label>
              <input v-model="instanceForm.name" class="input" placeholder="例如：主项目 / 备用窗口" />
            </div>
            <div>
              <label class="label">绑定账号</label>
              <select v-model="instanceForm.account_id" class="input">
                <option value="">不绑定</option>
                <option v-for="account in accounts" :key="account.id" :value="account.id">
                  {{ account.email }}
                </option>
              </select>
            </div>
          </div>
          <div class="grid gap-4 md:grid-cols-2">
            <div>
              <label class="label">工作目录</label>
              <input v-model="instanceForm.workspace_dir" class="input" placeholder="/path/to/project" />
            </div>
            <div>
              <label class="label">用户数据目录</label>
              <input v-model="instanceForm.user_data_dir" class="input" placeholder="/path/to/user-data" />
            </div>
          </div>
          <div class="grid gap-4 md:grid-cols-3">
            <div>
              <label class="label">状态</label>
              <select v-model="instanceForm.state" class="input">
                <option value="stopped">stopped</option>
                <option value="running">running</option>
                <option value="paused">paused</option>
              </select>
            </div>
            <div>
              <label class="label">PID</label>
              <input v-model="instanceForm.pid" class="input" placeholder="选填" />
            </div>
            <div class="flex items-end">
              <label class="flex items-center gap-2 text-sm text-gray-300">
                <input v-model="instanceForm.auto_start" type="checkbox" class="h-4 w-4 rounded border-gray-600 bg-gray-900 text-blue-500" />
                启动时自动拉起
              </label>
            </div>
          </div>
          <div>
            <label class="label">启动参数</label>
            <textarea v-model="instanceForm.launch_args" class="input min-h-24 font-mono text-xs" placeholder="例如：--new-window --disable-extensions" />
          </div>
          <div>
            <label class="label">备注</label>
            <textarea v-model="instanceForm.notes" class="input min-h-24" placeholder="记录这个实例的用途、项目或切号策略" />
          </div>
        </div>
        <div class="modal-footer">
          <button class="btn btn-secondary" @click="closeInstanceModal">取消</button>
          <button class="btn btn-primary" @click="saveInstance">保存</button>
        </div>
      </div>
    </div>

    <div v-if="showWakeupModal" class="modal-overlay" @click.self="closeWakeupModal">
      <div class="modal-content max-w-2xl">
        <div class="modal-header">
          <h3 class="text-white">{{ editingWakeup ? '编辑唤醒任务' : `新增 ${platform.label} 唤醒任务` }}</h3>
          <button class="text-gray-500 hover:text-white" @click="closeWakeupModal">✕</button>
        </div>
        <div class="modal-body space-y-4">
          <div class="grid gap-4 md:grid-cols-2">
            <div>
              <label class="label">任务名</label>
              <input v-model="wakeupForm.name" class="input" placeholder="例如：工作日前置唤醒" />
            </div>
            <div>
              <label class="label">绑定账号</label>
              <select v-model="wakeupForm.account_id" class="input">
                <option value="">不绑定</option>
                <option v-for="account in accounts" :key="account.id" :value="account.id">
                  {{ account.email }}
                </option>
              </select>
            </div>
          </div>
          <div class="grid gap-4 md:grid-cols-2">
            <div>
              <label class="label">模型</label>
              <input v-model="wakeupForm.model" class="input" placeholder="gpt-5.4 / gemini-2.5-pro" />
            </div>
            <div>
              <label class="label">调度类型</label>
              <select v-model="wakeupForm.schedule_type" class="input">
                <option value="daily">daily</option>
                <option value="weekly">weekly</option>
                <option value="interval">interval</option>
                <option value="manual">manual</option>
              </select>
            </div>
          </div>
          <div>
            <label class="label">调度表达</label>
            <input v-model="wakeupForm.schedule_value" class="input" placeholder="08:00 / Mon-Fri 09:00 / every 4h" />
          </div>
          <div>
            <label class="label">Prompt</label>
            <textarea v-model="wakeupForm.prompt" class="input min-h-28" placeholder="例如：hi / sync workspace / keep quota alive" />
          </div>
          <div class="flex items-center gap-2">
            <input v-model="wakeupForm.enabled" type="checkbox" class="h-4 w-4 rounded border-gray-600 bg-gray-900 text-blue-500" />
            <label class="text-sm text-gray-300">保存后立即启用</label>
          </div>
        </div>
        <div class="modal-footer">
          <button class="btn btn-secondary" @click="closeWakeupModal">取消</button>
          <button class="btn btn-primary" @click="saveWakeup">保存</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, inject, ref, watch } from 'vue'
import { cockpitAPI } from '@/api'
import PlatformIcon from '@/components/PlatformIcon.vue'
import { getPlatformMeta } from '@/lib/platforms'

const props = defineProps({
  platformId: {
    type: String,
    required: true,
  },
})

const notify = inject('notify')

const platform = computed(() => getPlatformMeta(props.platformId))
const supportsInstances = computed(() => !!platform.value?.supports.instances)
const supportsWakeup = computed(() => !!platform.value?.supports.wakeup)

const loading = ref(false)
const activeTab = ref('accounts')
const accountSearch = ref('')

const accounts = ref([])
const instances = ref([])
const wakeupTasks = ref([])

const showAccountModal = ref(false)
const showInstanceModal = ref(false)
const showWakeupModal = ref(false)
const showImportModal = ref(false)
const showOAuthModal = ref(false)

const editingAccount = ref(null)
const editingInstance = ref(null)
const editingWakeup = ref(null)

const accountForm = ref(createAccountForm())
const instanceForm = ref(createInstanceForm())
const wakeupForm = ref(createWakeupForm(props.platformId))
const importJson = ref('')

const oauthState = ref({
  loading: false,
  authUrl: '',
  userCode: '',
  deviceCode: '',
  loginId: '',
  callbackUrl: '',
  callbackSubmitting: false,
  manualCallback: false,
  callbackHint: '',
  waiting: false,
  success: false,
  error: '',
})

const supportsOAuth = computed(() => {
  // Platforms that support OAuth
  return ['antigravity', 'kiro', 'github-copilot', 'gemini'].includes(props.platformId)
})

const activeAccount = computed(() => accounts.value.find((item) => item.active))
const runningInstances = computed(() => instances.value.filter((item) => item.state === 'running'))
const enabledWakeupTasks = computed(() => wakeupTasks.value.filter((item) => item.enabled))

const filteredAccounts = computed(() => {
  const query = accountSearch.value.trim().toLowerCase()
  if (!query) return accounts.value
  return accounts.value.filter((account) => {
    const haystack = [
      account.email,
      account.display_name,
      account.plan,
      account.tag_name,
      account.status,
      account.notes,
    ]
      .filter(Boolean)
      .join(' ')
      .toLowerCase()
    return haystack.includes(query)
  })
})

watch(
  () => props.platformId,
  () => {
    accountSearch.value = ''
    if (!supportsInstances.value && activeTab.value === 'instances') activeTab.value = 'accounts'
    if (!supportsWakeup.value && activeTab.value === 'wakeup') activeTab.value = 'accounts'
    loadData()
  },
  { immediate: true }
)

async function loadData() {
  loading.value = true
  try {
    const requests = [
      cockpitAPI.listPlatformAccounts(props.platformId),
      supportsInstances.value ? cockpitAPI.listPlatformInstances(props.platformId) : Promise.resolve([]),
      supportsWakeup.value ? cockpitAPI.listWakeupTasks(props.platformId) : Promise.resolve([]),
    ]
    const [accountList, instanceList, taskList] = await Promise.all(requests)
    accounts.value = accountList
    instances.value = instanceList
    wakeupTasks.value = taskList
  } catch (error) {
    notify?.(error.message || '加载失败', 'error')
  } finally {
    loading.value = false
  }
}

function createAccountForm(account = null) {
  return {
    email: account?.email || '',
    display_name: account?.display_name || '',
    access_token: account?.access_token || '',
    refresh_token: account?.refresh_token || '',
    plan: account?.plan || '',
    status: account?.status || 'active',
    active: !!account?.active,
    quota_used: account?.quota_used ?? '',
    quota_limit: account?.quota_limit ?? '',
    quota_unit: account?.quota_unit || 'credits',
    tag_name: account?.tag_name || '',
    tag_color: account?.tag_color || '#3B82F6',
    notes: account?.notes || '',
  }
}

function createInstanceForm(instance = null) {
  return {
    name: instance?.name || '',
    account_id: instance?.account_id || '',
    workspace_dir: instance?.workspace_dir || '',
    user_data_dir: instance?.user_data_dir || '',
    launch_args: instance?.launch_args || '',
    state: instance?.state || 'stopped',
    pid: instance?.pid ?? '',
    auto_start: !!instance?.auto_start,
    notes: instance?.notes || '',
  }
}

function createWakeupForm(platformId, task = null) {
  return {
    platform: task?.platform || platformId,
    name: task?.name || '',
    account_id: task?.account_id || '',
    model: task?.model || '',
    prompt: task?.prompt || 'hi',
    schedule_type: task?.schedule_type || 'daily',
    schedule_value: task?.schedule_value || '08:00',
    enabled: task?.enabled ?? true,
  }
}

function openAccountModal(account = null) {
  editingAccount.value = account
  accountForm.value = createAccountForm(account)
  showAccountModal.value = true
}

function closeAccountModal() {
  editingAccount.value = null
  accountForm.value = createAccountForm()
  showAccountModal.value = false
}

async function saveAccount() {
  try {
    const payload = {
      email: accountForm.value.email.trim(),
      display_name: normalizeText(accountForm.value.display_name),
      access_token: normalizeText(accountForm.value.access_token),
      refresh_token: normalizeText(accountForm.value.refresh_token),
      plan: normalizeText(accountForm.value.plan),
      status: accountForm.value.status || 'active',
      active: !!accountForm.value.active,
      quota_used: normalizeNumber(accountForm.value.quota_used),
      quota_limit: normalizeNumber(accountForm.value.quota_limit),
      quota_unit: normalizeText(accountForm.value.quota_unit),
      tag_name: normalizeText(accountForm.value.tag_name),
      tag_color: normalizeText(accountForm.value.tag_color),
      notes: normalizeText(accountForm.value.notes),
    }
    if (!payload.email) {
      notify?.('请先填写邮箱', 'error')
      return
    }
    if (editingAccount.value) {
      await cockpitAPI.updatePlatformAccount(props.platformId, editingAccount.value.id, payload)
    } else {
      await cockpitAPI.addPlatformAccount(props.platformId, payload)
    }
    closeAccountModal()
    notify?.('账号已保存', 'success')
    await loadData()
  } catch (error) {
    notify?.(error.message || '保存失败', 'error')
  }
}

async function activateAccount(account) {
  try {
    await cockpitAPI.activatePlatformAccount(props.platformId, account.id)
    notify?.('已切换当前激活账号', 'success')
    await loadData()
  } catch (error) {
    notify?.(error.message || '切换失败', 'error')
  }
}

function openImportModal() {
  importJson.value = ''
  showImportModal.value = true
}

function closeImportModal() {
  importJson.value = ''
  showImportModal.value = false
}

async function importAccounts() {
  try {
    const payload = JSON.parse(importJson.value || '[]')
    if (!Array.isArray(payload)) {
      notify?.('导入内容必须是数组 JSON', 'error')
      return
    }
    const result = await cockpitAPI.importPlatformAccounts(props.platformId, payload)
    closeImportModal()
    notify?.(`已导入 ${result.imported || 0} 个账号`, 'success')
    await loadData()
  } catch (error) {
    notify?.(error.message || '导入失败', 'error')
  }
}

function downloadImportExample() {
  const payload = platformImportExamples[props.platformId] || platformImportExamples.default
  downloadJSON(payload, `${props.platformId}-accounts-example.json`)
  notify?.('示例 JSON 已下载', 'success')
}

async function exportAccounts() {
  try {
    const payload = await cockpitAPI.exportPlatformAccounts(props.platformId)
    downloadJSON(payload, `${props.platformId}-accounts.json`)
    notify?.('账号 JSON 已导出', 'success')
  } catch (error) {
    notify?.(error.message || '导出失败', 'error')
  }
}

async function deleteAccount(account) {
  if (!confirm(`确认删除账号 ${account.email} 吗？`)) return
  try {
    await cockpitAPI.deletePlatformAccount(props.platformId, account.id)
    notify?.('账号已删除', 'success')
    await loadData()
  } catch (error) {
    notify?.(error.message || '删除失败', 'error')
  }
}

// OAuth functions
function openOAuthModal() {
  showOAuthModal.value = true
  resetOAuthState()
}

function closeOAuthModal() {
  const pendingOAuth = {
    platformId: props.platformId,
    loginId: oauthState.value.loginId,
    deviceCode: oauthState.value.deviceCode,
    success: oauthState.value.success,
  }
  showOAuthModal.value = false
  if (!pendingOAuth.success && (pendingOAuth.loginId || pendingOAuth.deviceCode)) {
    cancelOAuth(pendingOAuth)
  }
  resetOAuthState()
}

function resetOAuthState() {
  oauthState.value = {
    loading: false,
    authUrl: '',
    userCode: '',
    deviceCode: '',
    loginId: '',
    callbackUrl: '',
    callbackSubmitting: false,
    manualCallback: false,
    callbackHint: '',
    waiting: false,
    success: false,
    error: '',
  }
}

async function startOAuth() {
  oauthState.value.loading = true
  oauthState.value.error = ''

  try {
    if (props.platformId === 'antigravity') {
      await startAntigravityOAuth()
    } else if (props.platformId === 'kiro') {
      await startKiroOAuth()
    } else if (props.platformId === 'github-copilot') {
      await startGitHubCopilotOAuth()
    } else if (props.platformId === 'gemini') {
      await startGeminiOAuth()
    } else {
      throw new Error('当前平台暂不支持 OAuth')
    }
  } catch (error) {
    oauthState.value.loading = false
    oauthState.value.error = error.message || '启动 OAuth 失败'
    notify?.(oauthState.value.error, 'error')
  }
}

async function startAntigravityOAuth() {
  const { antigravityOAuthAPI } = await import('@/api')
  const result = await antigravityOAuthAPI.startLogin()

  oauthState.value.authUrl = result.auth_url || result.verification_uri
  oauthState.value.loginId = result.login_id
  oauthState.value.loading = false
  oauthState.value.waiting = true
  oauthState.value.manualCallback = true
  oauthState.value.callbackHint = result.callback_url ? `回调监听地址：${result.callback_url}` : ''

  if (oauthState.value.authUrl) {
    window.open(oauthState.value.authUrl, '_blank')
  }

  completeAntigravityOAuth()
}

async function startKiroOAuth() {
  const { kiroOAuthAPI } = await import('@/api')
  const result = await kiroOAuthAPI.startLogin()

  oauthState.value.authUrl = result.verification_uri_complete || result.verification_uri
  oauthState.value.loginId = result.login_id
  oauthState.value.loading = false
  oauthState.value.waiting = true

  // Auto open browser
  if (oauthState.value.authUrl) {
    window.open(oauthState.value.authUrl, '_blank')
  }

  // Start polling for completion
  completeKiroOAuth()
}

async function startGitHubCopilotOAuth() {
  const { githubCopilotOAuthAPI } = await import('@/api')
  const result = await githubCopilotOAuthAPI.startLogin()

  oauthState.value.authUrl = result.verification_uri_complete || result.verification_uri
  oauthState.value.userCode = result.user_code
  oauthState.value.deviceCode = result.device_code
  oauthState.value.loading = false
  oauthState.value.waiting = true

  // Auto open browser
  if (oauthState.value.authUrl) {
    window.open(oauthState.value.authUrl, '_blank')
  }

  // Start polling for completion
  pollGitHubCopilotOAuth()
}

async function startGeminiOAuth() {
  const { geminiOAuthAPI } = await import('@/api')
  const result = await geminiOAuthAPI.startLogin()

  oauthState.value.authUrl = result.auth_url || result.verification_uri
  oauthState.value.loginId = result.login_id
  oauthState.value.loading = false
  oauthState.value.waiting = true
  oauthState.value.manualCallback = true
  oauthState.value.callbackHint = result.callback_url ? `回调监听地址：${result.callback_url}` : ''

  // Auto open browser
  if (oauthState.value.authUrl) {
    window.open(oauthState.value.authUrl, '_blank')
  }

  // Start polling for completion
  completeGeminiOAuth()
}

async function completeAntigravityOAuth() {
  if (!oauthState.value.loginId) return

  try {
    const { antigravityOAuthAPI } = await import('@/api')
    await antigravityOAuthAPI.completeLogin(oauthState.value.loginId, 600)

    oauthState.value.waiting = false
    oauthState.value.success = true
    notify?.('OAuth 授权成功，账号已添加', 'success')

    await loadData()

    setTimeout(() => {
      if (showOAuthModal.value) {
        closeOAuthModal()
      }
    }, 2000)
  } catch (error) {
    oauthState.value.waiting = false
    oauthState.value.error = error.message || 'OAuth 授权失败'
    notify?.(oauthState.value.error, 'error')
  }
}

async function completeKiroOAuth() {
  if (!oauthState.value.loginId) return

  try {
    const { kiroOAuthAPI } = await import('@/api')
    await kiroOAuthAPI.completeLogin(oauthState.value.loginId, 300)

    oauthState.value.waiting = false
    oauthState.value.success = true
    notify?.('OAuth 授权成功，账号已添加', 'success')

    await loadData()

    setTimeout(() => {
      if (showOAuthModal.value) {
        closeOAuthModal()
      }
    }, 2000)
  } catch (error) {
    oauthState.value.waiting = false
    oauthState.value.error = error.message || 'OAuth 授权失败'
    notify?.(oauthState.value.error, 'error')
  }
}

async function completeGeminiOAuth() {
  if (!oauthState.value.loginId) return

  try {
    const { geminiOAuthAPI } = await import('@/api')
    await geminiOAuthAPI.completeLogin(oauthState.value.loginId, 300)

    oauthState.value.waiting = false
    oauthState.value.success = true
    notify?.('OAuth 授权成功，账号已添加', 'success')

    await loadData()

    setTimeout(() => {
      if (showOAuthModal.value) {
        closeOAuthModal()
      }
    }, 2000)
  } catch (error) {
    oauthState.value.waiting = false
    oauthState.value.error = error.message || 'OAuth 授权失败'
    notify?.(oauthState.value.error, 'error')
  }
}

async function pollGitHubCopilotOAuth() {
  if (!oauthState.value.deviceCode) return

  const maxAttempts = 60 // 5 minutes with 5s interval
  let attempts = 0

  const poll = async () => {
    if (!showOAuthModal.value || !oauthState.value.waiting) return

    try {
      const { githubCopilotOAuthAPI } = await import('@/api')
      const result = await githubCopilotOAuthAPI.pollToken(oauthState.value.deviceCode)
      if (result?.status === 'pending') {
        attempts++
        if (attempts >= maxAttempts) {
          oauthState.value.waiting = false
          oauthState.value.error = '授权超时，请重试'
          notify?.(oauthState.value.error, 'error')
          return
        }
        setTimeout(poll, 5000)
        return
      }

      oauthState.value.waiting = false
      oauthState.value.success = true
      notify?.('OAuth 授权成功，账号已添加', 'success')

      await loadData()

      setTimeout(() => {
        if (showOAuthModal.value) {
          closeOAuthModal()
        }
      }, 2000)
    } catch (error) {
      attempts++
      if (attempts >= maxAttempts) {
        oauthState.value.waiting = false
        oauthState.value.error = '授权超时，请重试'
        notify?.(oauthState.value.error, 'error')
        return
      }

      // Continue polling if authorization_pending
      if (error.message?.includes('pending') || error.message?.includes('等待')) {
        setTimeout(poll, 5000)
      } else {
        oauthState.value.waiting = false
        oauthState.value.error = error.message || 'OAuth 授权失败'
        notify?.(oauthState.value.error, 'error')
      }
    }
  }

  poll()
}

async function completeOAuth() {
  // Deprecated - use platform-specific methods
  if (props.platformId === 'antigravity') {
    await completeAntigravityOAuth()
  } else if (props.platformId === 'kiro') {
    await completeKiroOAuth()
  } else if (props.platformId === 'github-copilot') {
    await pollGitHubCopilotOAuth()
  } else if (props.platformId === 'gemini') {
    await completeGeminiOAuth()
  }
}

async function cancelOAuth(pendingOAuth = null) {
  const platformId = pendingOAuth?.platformId || props.platformId
  const loginId = pendingOAuth?.loginId || oauthState.value.loginId

  try {
    if (platformId === 'antigravity' && loginId) {
      const { antigravityOAuthAPI } = await import('@/api')
      await antigravityOAuthAPI.cancelLogin(loginId)
    } else if (platformId === 'kiro' && loginId) {
      const { kiroOAuthAPI } = await import('@/api')
      await kiroOAuthAPI.cancelLogin(loginId)
    } else if (platformId === 'github-copilot') {
      const { githubCopilotOAuthAPI } = await import('@/api')
      await githubCopilotOAuthAPI.cancelLogin()
    } else if (platformId === 'gemini' && loginId) {
      const { geminiOAuthAPI } = await import('@/api')
      await geminiOAuthAPI.cancelLogin(loginId)
    }
  } catch (error) {
    console.error('Cancel OAuth failed:', error)
  }
}

async function submitOAuthCallback() {
  const callbackUrl = oauthState.value.callbackUrl.trim()
  if (!callbackUrl || !oauthState.value.loginId) return

  oauthState.value.callbackSubmitting = true
  oauthState.value.error = ''
  try {
    if (props.platformId === 'antigravity') {
      const { antigravityOAuthAPI } = await import('@/api')
      await antigravityOAuthAPI.submitCallback(oauthState.value.loginId, callbackUrl)
    } else if (props.platformId === 'gemini') {
      const { geminiOAuthAPI } = await import('@/api')
      await geminiOAuthAPI.submitCallback(oauthState.value.loginId, callbackUrl)
    } else {
      throw new Error('当前平台不支持手动回调')
    }
    notify?.('回调地址已提交，正在交换令牌', 'success')
  } catch (error) {
    oauthState.value.error = error.message || '提交回调失败'
    notify?.(oauthState.value.error, 'error')
  } finally {
    oauthState.value.callbackSubmitting = false
  }
}

function copyAuthUrl() {
  if (!oauthState.value.authUrl) return
  navigator.clipboard.writeText(oauthState.value.authUrl)
  notify?.('授权链接已复制', 'success')
}

function copyUserCode() {
  if (!oauthState.value.userCode) return
  navigator.clipboard.writeText(oauthState.value.userCode)
  notify?.('设备码已复制', 'success')
}

function openAuthUrl() {
  if (!oauthState.value.authUrl) return
  window.open(oauthState.value.authUrl, '_blank')
}

function openInstanceModal(instance = null) {
  editingInstance.value = instance
  instanceForm.value = createInstanceForm(instance)
  showInstanceModal.value = true
}

function closeInstanceModal() {
  editingInstance.value = null
  instanceForm.value = createInstanceForm()
  showInstanceModal.value = false
}

async function saveInstance() {
  try {
    const payload = {
      name: instanceForm.value.name.trim(),
      account_id: normalizeText(instanceForm.value.account_id),
      workspace_dir: normalizeText(instanceForm.value.workspace_dir),
      user_data_dir: normalizeText(instanceForm.value.user_data_dir),
      launch_args: normalizeText(instanceForm.value.launch_args),
      state: instanceForm.value.state || 'stopped',
      pid: normalizeInteger(instanceForm.value.pid),
      auto_start: !!instanceForm.value.auto_start,
      notes: normalizeText(instanceForm.value.notes),
    }
    if (!payload.name) {
      notify?.('请先填写实例名称', 'error')
      return
    }
    if (editingInstance.value) {
      await cockpitAPI.updatePlatformInstance(props.platformId, editingInstance.value.id, payload)
    } else {
      await cockpitAPI.addPlatformInstance(props.platformId, payload)
    }
    closeInstanceModal()
    notify?.('实例已保存', 'success')
    await loadData()
  } catch (error) {
    notify?.(error.message || '保存失败', 'error')
  }
}

async function exportInstances() {
  try {
    const payload = await cockpitAPI.exportPlatformInstances(props.platformId)
    downloadJSON(payload, `${props.platformId}-instances.json`)
    notify?.('实例 JSON 已导出', 'success')
  } catch (error) {
    notify?.(error.message || '导出失败', 'error')
  }
}

async function setInstanceState(instance, state) {
  try {
    await cockpitAPI.updatePlatformInstanceState(props.platformId, instance.id, state)
    notify?.(`实例状态已切换为 ${state}`, 'success')
    await loadData()
  } catch (error) {
    notify?.(error.message || '状态切换失败', 'error')
  }
}

async function deleteInstance(instance) {
  if (!confirm(`确认删除实例 ${instance.name} 吗？`)) return
  try {
    await cockpitAPI.deletePlatformInstance(props.platformId, instance.id)
    notify?.('实例已删除', 'success')
    await loadData()
  } catch (error) {
    notify?.(error.message || '删除失败', 'error')
  }
}

function openWakeupModal(task = null) {
  editingWakeup.value = task
  wakeupForm.value = createWakeupForm(props.platformId, task)
  showWakeupModal.value = true
}

function closeWakeupModal() {
  editingWakeup.value = null
  wakeupForm.value = createWakeupForm(props.platformId)
  showWakeupModal.value = false
}

async function saveWakeup() {
  try {
    const payload = {
      platform: props.platformId,
      name: wakeupForm.value.name.trim(),
      account_id: normalizeText(wakeupForm.value.account_id),
      model: normalizeText(wakeupForm.value.model),
      prompt: normalizeText(wakeupForm.value.prompt),
      schedule_type: wakeupForm.value.schedule_type || 'daily',
      schedule_value: wakeupForm.value.schedule_value?.trim() || '08:00',
      enabled: !!wakeupForm.value.enabled,
    }
    if (!payload.name) {
      notify?.('请先填写任务名', 'error')
      return
    }
    if (editingWakeup.value) {
      await cockpitAPI.updateWakeupTask(editingWakeup.value.id, payload)
    } else {
      await cockpitAPI.addWakeupTask(payload)
    }
    closeWakeupModal()
    notify?.('唤醒任务已保存', 'success')
    await loadData()
  } catch (error) {
    notify?.(error.message || '保存失败', 'error')
  }
}

async function toggleWakeup(task) {
  try {
    await cockpitAPI.toggleWakeupTask(task.id)
    notify?.(task.enabled ? '任务已停用' : '任务已启用', 'success')
    await loadData()
  } catch (error) {
    notify?.(error.message || '操作失败', 'error')
  }
}

async function deleteWakeup(task) {
  if (!confirm(`确认删除唤醒任务 ${task.name} 吗？`)) return
  try {
    await cockpitAPI.deleteWakeupTask(task.id)
    notify?.('唤醒任务已删除', 'success')
    await loadData()
  } catch (error) {
    notify?.(error.message || '删除失败', 'error')
  }
}

function normalizeText(value) {
  const text = typeof value === 'string' ? value.trim() : ''
  return text || null
}

function normalizeNumber(value) {
  if (value === '' || value == null) return null
  const num = Number(value)
  return Number.isFinite(num) ? num : null
}

function normalizeInteger(value) {
  if (value === '' || value == null) return null
  const num = Number(value)
  return Number.isFinite(num) ? Math.trunc(num) : null
}

function formatDateTime(value) {
  if (!value) return '未记录'
  return new Date(value).toLocaleString('zh-CN')
}

function quotaText(account) {
  if (account.quota_used == null && account.quota_limit == null) return '未记录'
  const unit = account.quota_unit || 'quota'
  if (account.quota_limit == null) return `${account.quota_used ?? 0} ${unit}`
  return `${account.quota_used ?? 0} / ${account.quota_limit} ${unit}`
}

function accountLabel(accountId) {
  if (!accountId) return '未绑定'
  const account = accounts.value.find((item) => item.id === accountId)
  return account?.email || '账号已移除'
}

function scheduleText(task) {
  const type = task.schedule_type || 'daily'
  const value = task.schedule_value || '未定义'
  return `${type} · ${value}`
}

const platformImportExamples = {
  antigravity: [
    {
      email: 'antigravity-user@example.com',
      display_name: 'Antigravity User',
      access_token: 'ya29.access_token_here',
      refresh_token: 'refresh_token_here',
      plan: 'pro',
      active: true,
      metadata_json: '{"project_id":"your-project-id","auth_source":"manual"}',
    },
  ],
  'github-copilot': [
    {
      email: 'github-user@example.com',
      display_name: 'GitHub User',
      access_token: 'gho_github_access_token_here',
      plan: 'copilot',
      active: true,
      metadata_json: '{"github_login":"your-github-login","github_access_token":"gho_github_access_token_here","auth_source":"manual"}',
    },
  ],
  kiro: [
    {
      email: 'kiro-user@example.com',
      display_name: 'Kiro User',
      access_token: 'kiro_access_token_here',
      refresh_token: 'kiro_refresh_token_here',
      plan: 'builder',
      active: true,
      metadata_json: '{"profile_arn":"arn:aws:sso:::profile/your-profile","idc_region":"us-east-1","auth_source":"manual"}',
    },
  ],
  gemini: [
    {
      email: 'gemini-user@gmail.com',
      display_name: 'Gemini User',
      access_token: 'ya29.gemini_access_token_here',
      refresh_token: 'gemini_refresh_token_here',
      plan: 'free-tier',
      active: true,
      metadata_json: '{"project_id":"your-google-cloud-project","auth_source":"manual"}',
    },
  ],
  default: [
    {
      email: 'user@example.com',
      display_name: 'Example User',
      access_token: 'access_token_here',
      refresh_token: 'refresh_token_here',
      plan: 'pro',
      active: true,
      metadata_json: '{"auth_source":"manual"}',
    },
  ],
}

function downloadJSON(payload, filename) {
  const text = JSON.stringify(payload, null, 2)
  const blob = new Blob([text], { type: 'application/json;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  link.click()
  URL.revokeObjectURL(url)
}
</script>
