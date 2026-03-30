<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/stores.svelte';
	import {
		goalsAPI,
		type GoalDashboard,
		type GoalDashboardItem,
		type GoalPeriodDetail,
		type GoalYearSummary
	} from '$lib/api';
	import { formatLongDate, getTodayDateString, getYesterdayDateString } from '$lib/date';

	type GoalFormState = {
		reactivateId: string;
		name: string;
		description: string;
		unit: string;
		annualTarget: string | number;
		weeklyTarget: string | number;
	};

	type DeleteState = {
		goal: GoalDashboardItem | null;
		nameInput: string;
		error: string;
	};

	type CheckinDraft = {
		checked: boolean;
		quantity: string;
		originalChecked: boolean;
		originalQuantity: string;
	};

	type MonthlyAggregate = {
		goalId: number;
		name: string;
		unit: string;
		total: number;
	};

	function getCurrentYear() {
		return new Date().getFullYear();
	}

	function getStartOfCurrentYearDateString() {
		return `${getCurrentYear()}-01-01`;
	}

	function getDefaultCheckinDateString() {
		const yesterday = getYesterdayDateString();
		return yesterday < getStartOfCurrentYearDateString() ? getStartOfCurrentYearDateString() : yesterday;
	}

	function shiftDate(dateStr: string, delta: number): string {
		const [year, month, day] = dateStr.split('-').map(Number);
		const next = new Date(year, month - 1, day);
		next.setDate(next.getDate() + delta);
		const nextYear = next.getFullYear();
		const nextMonth = `${next.getMonth() + 1}`.padStart(2, '0');
		const nextDay = `${next.getDate()}`.padStart(2, '0');
		return `${nextYear}-${nextMonth}-${nextDay}`;
	}

	let dashboard = $state<GoalDashboard | null>(null);
	let previousYearSummary = $state<GoalYearSummary | null>(null);
	let previousYear = $state(getCurrentYear() - 1);
	let isLoading = $state(true);
	let isRefreshing = $state(false);
	let isSaving = $state(false);
	let isMutating = $state(false);
	let isYearSummaryLoading = $state(false);
	let error = $state('');
	let anchorDate = $state(getTodayDateString());
	let checkinDate = $state(getDefaultCheckinDateString());
	let showFormModal = $state(false);
	let showDeleteModal = $state(false);
	let isEditing = $state(false);
	let editingGoalId = $state<number | null>(null);
	let formError = $state('');
	let quantityDrafts = $state<Record<number, string>>({});
	let checkinDrafts = $state<Record<number, CheckinDraft>>({});
	let goalForm = $state<GoalFormState>(emptyGoalForm());
	let deleteState = $state<DeleteState>({ goal: null, nameInput: '', error: '' });

	onMount(() => {
		void initPage();
	});

	$effect(() => {
		if (!showFormModal && !showDeleteModal) {
			return;
		}

		const handleKeydown = (event: KeyboardEvent) => {
			if (event.key === 'Escape') {
				if (showDeleteModal) {
					closeDeleteModal();
					return;
				}
				closeFormModal();
			}
		};

		window.addEventListener('keydown', handleKeydown);
		return () => window.removeEventListener('keydown', handleKeydown);
	});

	async function initPage() {
		await auth.ensureInitialized();
		if (!auth.isLoggedIn) {
			goto('/');
			return;
		}
		await loadDashboard();
		await loadPreviousYearSummary(previousYear);
	}

	async function loadDashboard(options?: { silent?: boolean }) {
		const silent = options?.silent ?? false;
		if (dashboard && silent) {
			isRefreshing = true;
		} else {
			isLoading = true;
		}
		error = '';

		try {
			const result = await goalsAPI.dashboard({
				date: anchorDate,
				checkin_date: checkinDate
			});
			dashboard = result;
			checkinDate = result.checkin_date;
			initializeQuantityDrafts(result.goals);
		} catch (err) {
			error = err instanceof Error ? err.message : '加载目标面板失败';
			if (!dashboard) {
				dashboard = null;
			}
		} finally {
			isLoading = false;
			isRefreshing = false;
		}
	}

	async function loadPreviousYearSummary(year: number) {
		isYearSummaryLoading = true;
		try {
			const result = await goalsAPI.yearSummary(year);
			previousYearSummary = result.has_records ? result : null;
		} catch (err) {
			error = err instanceof Error ? err.message : '加载往年统计失败';
			previousYearSummary = null;
		} finally {
			isYearSummaryLoading = false;
		}
	}

	function initializeQuantityDrafts(goals: GoalDashboardItem[]) {
		const nextDrafts: Record<number, string> = {};
		const nextCheckinDrafts: Record<number, CheckinDraft> = {};
		for (const goal of goals) {
			const quantity = goal.checkin_record?.quantity?.toString() ?? '';
			const checked = !!goal.checkin_record;
			nextCheckinDrafts[goal.id] = {
				checked,
				quantity,
				originalChecked: checked,
				originalQuantity: quantity
			};
			if (goal.unit) {
				nextDrafts[goal.id] = quantity;
			}
		}
		quantityDrafts = nextDrafts;
		checkinDrafts = nextCheckinDrafts;
	}

	function emptyGoalForm(): GoalFormState {
		return {
			reactivateId: '',
			name: '',
			description: '',
			unit: '',
			annualTarget: '',
			weeklyTarget: ''
		};
	}

	function openCreateModal() {
		goalForm = emptyGoalForm();
		formError = '';
		showFormModal = true;
		isEditing = false;
		editingGoalId = null;
	}

	function openEditModal(goal: GoalDashboardItem) {
		goalForm = {
			reactivateId: '',
			name: goal.name,
			description: goal.description,
			unit: goal.unit,
			annualTarget: goal.annual_target?.toString() ?? '',
			weeklyTarget: goal.weekly_target?.toString() ?? ''
		};
		formError = '';
		showFormModal = true;
		isEditing = true;
		editingGoalId = goal.id;
	}

	function closeFormModal() {
		showFormModal = false;
		formError = '';
	}

	function openDeleteModal(goal: GoalDashboardItem) {
		deleteState = { goal, nameInput: '', error: '' };
		showDeleteModal = true;
	}

	function closeDeleteModal() {
		showDeleteModal = false;
		deleteState = { goal: null, nameInput: '', error: '' };
	}

	async function handleGoalSubmit(event: Event) {
		event.preventDefault();
		formError = '';
		isSaving = true;

		try {
			const reactivateId = goalForm.reactivateId ? Number(goalForm.reactivateId) : undefined;
			const reactivatedGoal = reactivateId ? dashboard?.inactive_goals.find((goal) => goal.id === reactivateId) ?? null : null;
			const annualTarget = parseOptionalNumber(goalForm.annualTarget);
			const weeklyTarget = parseOptionalInt(goalForm.weeklyTarget);
			const payload = {
				reactivate_id: reactivateId,
				name: (reactivatedGoal?.name ?? goalForm.name).trim(),
				description: (reactivatedGoal?.description ?? goalForm.description).trim() || undefined,
				unit: (reactivatedGoal?.unit ?? goalForm.unit).trim() || undefined,
				annual_target: isEditing ? (annualTarget ?? 0) : annualTarget,
				weekly_target: weeklyTarget
			};

			if (isEditing && editingGoalId !== null) {
				await goalsAPI.update(editingGoalId, payload);
			} else {
				await goalsAPI.create(payload);
			}

			closeFormModal();
			await loadDashboard({ silent: true });
			await loadPreviousYearSummary(previousYear);
		} catch (err) {
			formError = err instanceof Error ? err.message : '保存目标失败';
		} finally {
			isSaving = false;
		}
	}

	async function confirmDeleteGoal() {
		if (!deleteState.goal) {
			return;
		}
		if (deleteState.nameInput.trim() !== deleteState.goal.name) {
			deleteState = { ...deleteState, error: '请输入完全一致的目标名称后再删除' };
			return;
		}

		isMutating = true;
		try {
			await goalsAPI.delete(deleteState.goal.id);
			closeDeleteModal();
			await loadDashboard({ silent: true });
			await loadPreviousYearSummary(previousYear);
		} catch (err) {
			deleteState = {
				...deleteState,
				error: err instanceof Error ? err.message : '删除目标失败'
			};
		} finally {
			isMutating = false;
		}
	}

	function toggleCheckin(goal: GoalDashboardItem) {
		const current = checkinDrafts[goal.id];
		if (!current) return;
		checkinDrafts = {
			...checkinDrafts,
			[goal.id]: {
				...current,
				checked: !current.checked
			}
		};
	}

	function updateQuantityDraft(goalId: number, value: string) {
		quantityDrafts = { ...quantityDrafts, [goalId]: value };
		const current = checkinDrafts[goalId];
		if (!current) return;
		checkinDrafts = {
			...checkinDrafts,
			[goalId]: {
				...current,
				quantity: value,
				checked: value.trim() !== ''
			}
		};
	}

	async function saveCheckins() {
		isMutating = true;
		try {
			for (const goal of dashboard?.goals ?? []) {
				const draft = checkinDrafts[goal.id];
				if (!draft) {
					continue;
				}

				const changed = draft.checked !== draft.originalChecked || draft.quantity !== draft.originalQuantity;
				if (!changed) {
					continue;
				}

				if (goal.unit) {
					if (!draft.checked || draft.quantity.trim() === '') {
						if (draft.originalChecked) {
							await goalsAPI.deleteRecord(goal.id, checkinDate);
						}
						continue;
					}

					const quantity = Number(draft.quantity);
					if (!Number.isFinite(quantity) || quantity <= 0) {
						throw new Error(`请输入目标“${goal.name}”的大于 0 的数量`);
					}
					await goalsAPI.upsertRecord(goal.id, checkinDate, { quantity });
					continue;
				}

				if (draft.checked) {
					await goalsAPI.upsertRecord(goal.id, checkinDate);
				} else if (draft.originalChecked) {
					await goalsAPI.deleteRecord(goal.id, checkinDate);
				}
			}

			await loadDashboard({ silent: true });
			await loadPreviousYearSummary(previousYear);
		} catch (err) {
			error = err instanceof Error ? err.message : '保存打卡失败';
		} finally {
			isMutating = false;
		}
	}

	function parseOptionalNumber(value: string | number): number | undefined {
		if (value === '') return undefined;
		const parsed = typeof value === 'number' ? value : Number(value.trim());
		return Number.isFinite(parsed) ? parsed : undefined;
	}

	function parseOptionalInt(value: string | number): number | undefined {
		if (value === '') return undefined;
		const parsed = typeof value === 'number' ? value : Number.parseInt(value.trim(), 10);
		return Number.isFinite(parsed) ? parsed : undefined;
	}

	function handleReactivateChange(value: string) {
		goalForm.reactivateId = value;
		if (!value || !dashboard) {
			goalForm = { ...goalForm, reactivateId: value };
			return;
		}

		const inactiveGoal = dashboard.inactive_goals.find((goal) => goal.id === Number(value));
		if (!inactiveGoal) {
			return;
		}

		goalForm = {
			reactivateId: value,
			name: inactiveGoal.name,
			description: inactiveGoal.description,
			unit: inactiveGoal.unit,
			annualTarget: inactiveGoal.annual_target?.toString() ?? '',
			weeklyTarget: inactiveGoal.weekly_target?.toString() ?? ''
		};
	}

	function switchCheckinDate(delta: number) {
		const nextDate = shiftDate(checkinDate, delta);
		if (nextDate < getStartOfCurrentYearDateString()) {
			return;
		}
		if (nextDate > getTodayDateString()) {
			return;
		}
		checkinDate = nextDate;
		void loadDashboard({ silent: true });
	}

	function loadEarlierYear() {
		previousYear -= 1;
		void loadPreviousYearSummary(previousYear);
	}

	function loadLaterYear() {
		if (previousYear >= getCurrentYear() - 1) {
			return;
		}
		previousYear += 1;
		void loadPreviousYearSummary(previousYear);
	}

	function getProgressStyle(percent: number | null) {
		return `width: ${Math.max(0, Math.min(percent ?? 0, 100))}%`;
	}

	function formatAnnualSummary(goal: GoalDashboardItem) {
		if (goal.unit) {
			if (goal.annual_target !== null) {
				return `${goal.annual_quantity_total}/${goal.annual_target}${goal.unit}`;
			}
			return `本年 ${goal.annual_quantity_total}${goal.unit}`;
		}

		if (goal.annual_target !== null) {
			return `${goal.annual_completed_count}/${goal.annual_target} 次`;
		}
		return `本年 ${goal.annual_completed_count} 次`;
	}

	function formatPeriodItem(item: GoalPeriodDetail['items'][number]) {
		if (item.unit) {
			return item.quantity !== null ? `${item.quantity}${item.unit}` : '已记录';
		}
		return item.is_completed ? '已完成' : '未完成';
	}

	function buildMonthlyAggregates(details: GoalPeriodDetail[]): MonthlyAggregate[] {
		const aggregates = new Map<number, MonthlyAggregate>();
		for (const detail of details) {
			for (const item of detail.items) {
				const current = aggregates.get(item.goal_id) ?? {
					goalId: item.goal_id,
					name: item.name,
					unit: item.unit,
					total: 0
				};
				current.total += item.unit ? item.quantity ?? 0 : item.is_completed ? 1 : 0;
				aggregates.set(item.goal_id, current);
			}
		}

		return Array.from(aggregates.values()).sort((a, b) => b.total - a.total);
	}

	function hasPendingCheckinChanges() {
		return Object.values(checkinDrafts).some((draft) => draft.checked !== draft.originalChecked || draft.quantity !== draft.originalQuantity);
	}

	const hasGoals = $derived((dashboard?.goals.length ?? 0) > 0);
	const monthlyAggregates = $derived(buildMonthlyAggregates(dashboard?.month_details ?? []));
</script>

<div class="goals-page">
	{#if error}
		<div class="alert variant-soft-error">
			<p>{error}</p>
		</div>
	{/if}

	{#if isLoading && !dashboard}
		<div class="loading-state card">
			<p class="text-surface-500">目标面板加载中...</p>
		</div>
	{:else if dashboard}
		<div class:page-refreshing={isRefreshing} class="goals-content">
			{#if isRefreshing}
				<div class="refresh-indicator" aria-live="polite">正在更新视图...</div>
			{/if}

			<section class="goals-hero card">
				<div>
					<p class="goals-kicker">Goals Dashboard</p>
					<h1>目标</h1>
					<p class="goals-copy">先完成今天最需要的打卡，再顺手回看今年的累计与节奏，目标管理尽量退到次要位置。</p>
				</div>
				<div class="hero-actions">
					<div class="hero-date-chip">
						<span>今天</span>
						<strong>{anchorDate}</strong>
					</div>
				</div>
			</section>

			{#if !hasGoals}
				<section class="empty-goals card">
					<h2>还没有目标</h2>
					<p>先创建几个想长期坚持的目标，再开始打卡与年度回顾。</p>
					<button class="btn variant-filled-primary" type="button" onclick={openCreateModal}>创建第一个目标</button>
				</section>
			{:else}
				<section class="checkin-panel card">
					<div class="section-head">
						<div>
							<p class="section-kicker">Check-in</p>
							<h2>{formatLongDate(checkinDate)}</h2>
							<p class="section-copy">默认从昨天开始，左右逐天调整。可回改今年内的记录，但不允许跨到去年或明天。</p>
						</div>
					<div class="date-stepper">
						<button class="btn variant-soft-surface" type="button" onclick={() => switchCheckinDate(-1)} disabled={checkinDate <= getStartOfCurrentYearDateString()}>前一天</button>
						<div class="checkin-summary-chip">
							<strong>{dashboard.goals.filter((goal) => goal.checkin_record).length}</strong>
							<span>/ {dashboard.goals.length} 已记录</span>
						</div>
						<button class="btn variant-soft-surface" type="button" onclick={() => switchCheckinDate(1)} disabled={checkinDate >= getTodayDateString()}>后一天</button>
					</div>
				</div>

					<div class="checkin-list">
						{#each dashboard.goals as goal (goal.id)}
							<div class="checkin-row">
								<div>
									<p class="checkin-goal-name">{goal.name}</p>
									{#if goal.description}
										<p class="goal-description">{goal.description}</p>
									{/if}
									<div class="checkin-tags">
										{#if goal.unit}
											<span class="goal-inline-pill">单位：{goal.unit}</span>
										{/if}
										{#if goal.weekly_target !== null}
											<span class="goal-inline-pill">周目标 {goal.weekly_target} 次</span>
										{/if}
										{#if goal.annual_target !== null}
											<span class="goal-inline-pill subtle-pill">年度目标 {goal.annual_target}{goal.unit || ' 次'}</span>
										{/if}
									</div>
								</div>

								{#if goal.unit}
									<div class="checkin-quantity">
										<input class="input quantity-input" type="number" min="0" step="0.1" value={quantityDrafts[goal.id] ?? ''} oninput={(event) => updateQuantityDraft(goal.id, (event.currentTarget as HTMLInputElement).value)} placeholder={`输入${goal.unit}`} />
									</div>
								{:else}
									<button class:checkin-done={!!checkinDrafts[goal.id]?.checked} class="checkin-toggle" type="button" onclick={() => toggleCheckin(goal)} disabled={isMutating}>
										<span>{checkinDrafts[goal.id]?.checked ? '已完成' : '待记录'}</span>
										<strong>{goal.weekly_target !== null ? `${goal.current_week_completed_count}/${goal.weekly_target}` : goal.checkin_record ? '✓' : '+'}</strong>
									</button>
								{/if}
							</div>
						{/each}
					</div>

					<div class="checkin-actions">
						<button class="btn variant-filled-primary" type="button" onclick={saveCheckins} disabled={isMutating || !hasPendingCheckinChanges()}>
							{isMutating ? '保存中...' : '保存本次打卡'}
						</button>
					</div>
				</section>

				<section class="stats-panel">
					<div class="overview-strip">
						<article class="overview-stat card">
							<p>活跃目标</p>
							<strong>{dashboard.goals.length}</strong>
							<span>当前仍在推进的目标清单</span>
						</article>
						<article class="overview-stat card">
							<p>今日打卡</p>
							<strong>{dashboard.today_completed_count}</strong>
							<span>今天已完成的目标数量</span>
						</article>
						<article class="overview-stat card">
							<p>年度打卡</p>
							<strong>{dashboard.annual_checkin_total}</strong>
							<span>{dashboard.range_start_date} 到 {dashboard.range_end_date}</span>
						</article>
					</div>

					<section class="goal-progress-grid">
						{#each dashboard.goals as goal (goal.id)}
							<article class:goal-card-strong={!!goal.checkin_record} class="goal-progress-card card">
								<div class="goal-progress-head">
									<div>
										<h2>{goal.name}</h2>
										{#if goal.description}
											<p class="goal-description">{goal.description}</p>
										{/if}
									</div>
									<div class="goal-card-actions">
										<button class="btn btn-sm variant-soft-surface" type="button" onclick={() => openEditModal(goal)}>编辑</button>
										<button class="btn btn-sm btn-quiet" type="button" onclick={() => openDeleteModal(goal)} disabled={isMutating}>删除</button>
									</div>
								</div>

								<div class="goal-progress-bar">
									<span class="goal-progress-fill" style={getProgressStyle(goal.annual_progress_percent ?? goal.current_week_progress_percent ?? 0)}></span>
								</div>

								<div class="goal-progress-meta">
									<strong>{formatAnnualSummary(goal)}</strong>
									{#if goal.annual_remaining_value !== null}
										<span>还差 {goal.annual_remaining_value}{goal.unit || ' 次'}</span>
									{:else if goal.current_week_progress_percent !== null && goal.weekly_target !== null}
										<span>本周 {goal.current_week_completed_count}/{goal.weekly_target} · {goal.current_week_progress_percent}%</span>
									{:else}
										<span>本年已累计 {goal.unit ? `${goal.annual_quantity_total}${goal.unit}` : `${goal.annual_completed_count} 次`}</span>
									{/if}
								</div>

								<div class="goal-card-footer">
									{#if goal.weekly_target !== null}
										<span class="goal-inline-pill">周目标 {goal.weekly_target} 次</span>
									{/if}
									{#if goal.unit}
										<span class="goal-inline-pill">单位：{goal.unit}</span>
									{/if}
									<span class="goal-inline-pill subtle-pill">今年打卡 {goal.annual_completed_count} 次</span>
								</div>
							</article>
						{/each}
					</section>

					<section class="detail-grid">
						<div class="detail-panel card">
							<div class="detail-head">
								<div>
									<p class="section-kicker">Week Detail</p>
									<h2>这周记录</h2>
									<p class="section-copy">快速回看这周具体记录了哪些任务。</p>
								</div>
							</div>
							{#if dashboard.week_details.length === 0}
								<p class="text-surface-500">这周还没有记录。</p>
							{:else}
								<div class="period-detail-list">
									{#each dashboard.week_details as detail (detail.date)}
										<div class="period-day-block">
											<div class="period-day-head">
												<strong>{formatLongDate(detail.date)}</strong>
												<span>{detail.completed_goals} 项</span>
											</div>
											{#each detail.items as item (item.goal_id)}
												<div class="period-detail-item">
													<span>{item.name}</span>
													<small>{formatPeriodItem(item)}</small>
												</div>
											{/each}
										</div>
									{/each}
								</div>
							{/if}
						</div>

						<div class="detail-panel card">
							<div class="detail-head">
								<div>
									<p class="section-kicker">Month Detail</p>
									<h2>本月记录</h2>
									<p class="section-copy">按目标聚合本月累计，不再展开到每天的细节。</p>
								</div>
							</div>
							{#if monthlyAggregates.length === 0}
								<p class="text-surface-500">本月还没有记录。</p>
							{:else}
								<div class="period-detail-list">
									{#each monthlyAggregates as item (item.goalId)}
										<div class="period-day-block">
											<div class="period-day-head">
												<strong>{item.name}</strong>
												<span>{item.unit ? `${item.total}${item.unit}` : `${item.total} 次`}</span>
											</div>
											<div class="period-detail-item">
												<span>本月累计</span>
												<small>{item.unit ? `${item.total}${item.unit}` : `${item.total} 次`}</small>
											</div>
										</div>
									{/each}
								</div>
							{/if}
						</div>
					</section>
				</section>

				<section class="create-panel card">
					<div>
						<p class="section-kicker">New Goal</p>
						<h2>新增目标</h2>
						<p class="section-copy">新增是低频动作，放在页面最后。目标名称必填，备注、单位、年度目标和每周目标都可选。</p>
					</div>
					<button class="btn variant-filled-primary" type="button" onclick={openCreateModal}>新增目标</button>
				</section>

				{#if isYearSummaryLoading}
					<section class="history-panel card">
						<p class="text-surface-500">往年统计加载中...</p>
					</section>
				{:else if previousYearSummary}
					<section class="history-panel card">
						<div class="history-head">
							<div>
								<p class="section-kicker">History</p>
								<h2>{previousYearSummary.year} 年统计</h2>
								<p class="section-copy">如果往年有记录，就保留一个轻量年度回顾，支持继续往前翻。</p>
							</div>
							<div class="calendar-nav">
								<button class="btn variant-soft-surface" type="button" onclick={loadEarlierYear}>更早一年</button>
								<button class="btn variant-soft-surface" type="button" onclick={loadLaterYear} disabled={previousYear >= getCurrentYear() - 1}>更近一年</button>
							</div>
						</div>
						<div class="overview-strip history-stats">
							<article class="overview-stat card">
								<p>累计打卡</p>
								<strong>{previousYearSummary.total_checkins}</strong>
								<span>{previousYearSummary.start_date} 到 {previousYearSummary.end_date}</span>
							</article>
							<article class="overview-stat card">
								<p>记录目标</p>
								<strong>{previousYearSummary.recorded_goal_count}</strong>
								<span>这一年里留下记录的任务数</span>
							</article>
							<article class="overview-stat card">
								<p>记录天数</p>
								<strong>{previousYearSummary.recorded_days}</strong>
								<span>这一年实际留下记录的日期数</span>
							</article>
						</div>
					</section>
				{/if}
			{/if}
		</div>
	{/if}

	{#if showFormModal}
		<div class="modal-backdrop" role="presentation">
			<div class="modal-card card" role="dialog" aria-modal="true" aria-labelledby="goal-modal-title">
				<div class="modal-head">
					<div>
						<p class="section-kicker">{isEditing ? 'Edit Goal' : 'New Goal'}</p>
						<h2 id="goal-modal-title">{isEditing ? '编辑目标' : '新增目标'}</h2>
					</div>
					<button class="btn btn-quiet" type="button" onclick={closeFormModal}>关闭</button>
				</div>

				{#if formError}
					<div class="alert variant-soft-error">
						<p>{formError}</p>
					</div>
				{/if}

				<form onsubmit={handleGoalSubmit} class="goal-form">
					{#if !isEditing && (dashboard?.inactive_goals.length ?? 0) > 0}
						<label class="label">
							<span>重新启用已删除目标（可选）</span>
							<select class="input" value={goalForm.reactivateId} onchange={(event) => handleReactivateChange((event.currentTarget as HTMLSelectElement).value)}>
								<option value="">新增全新目标</option>
								{#each dashboard?.inactive_goals ?? [] as inactiveGoal (inactiveGoal.id)}
									<option value={inactiveGoal.id.toString()}>{inactiveGoal.name}</option>
								{/each}
							</select>
						</label>
					{/if}

					<label class="label">
						<span>目标名称</span>
						<input class="input" type="text" bind:value={goalForm.name} placeholder="例如：跑步、冥想、读书" required disabled={!!goalForm.reactivateId && !isEditing} />
					</label>

					<label class="label">
						<span>备注说明（可选）</span>
						<textarea class="input textarea-input" bind:value={goalForm.description} placeholder="例如：只记录晨跑；周末补记也可以" disabled={!!goalForm.reactivateId && !isEditing}></textarea>
					</label>

					<label class="label">
						<span>单位（可选，填写后打卡需输入数量）</span>
						<input class="input" type="text" bind:value={goalForm.unit} placeholder="km / 本 / 页" disabled={!!goalForm.reactivateId && !isEditing} />
					</label>

					<label class="label">
						<span>年度目标（可选）</span>
						<input class="input" type="number" min="0" step="0.1" bind:value={goalForm.annualTarget} placeholder="例如 500" />
					</label>

					<label class="label">
						<span>每周目标次数（可选）</span>
						<input class="input" type="number" min="1" step="1" bind:value={goalForm.weeklyTarget} placeholder="例如 3" />
					</label>

					<div class="modal-actions">
						<button type="button" class="btn variant-soft-surface" onclick={closeFormModal}>取消</button>
						<button type="submit" class="btn variant-filled-primary" disabled={isSaving}>{isSaving ? '保存中...' : '保存目标'}</button>
					</div>
				</form>
			</div>
		</div>
	{/if}

	{#if showDeleteModal && deleteState.goal}
		<div class="modal-backdrop" role="presentation" onclick={(event) => event.target === event.currentTarget && closeDeleteModal()}>
			<div class="modal-card card delete-modal" role="dialog" aria-modal="true" aria-labelledby="goal-delete-title">
				<div class="modal-head">
					<div>
						<p class="section-kicker">Delete Goal</p>
						<h2 id="goal-delete-title">删除目标</h2>
					</div>
					<button class="btn btn-quiet" type="button" onclick={closeDeleteModal}>关闭</button>
				</div>
				<p class="section-copy">这是软删除，历史记录会保留。未来如果重新启用这个目标，旧记录会再次出现。</p>
				<p class="delete-goal-name">请输入 <strong>{deleteState.goal.name}</strong> 确认删除。</p>
				<input class="input" type="text" bind:value={deleteState.nameInput} placeholder="输入目标名称" />
				{#if deleteState.error}
					<div class="alert variant-soft-error">
						<p>{deleteState.error}</p>
					</div>
				{/if}
				<div class="modal-actions">
					<button type="button" class="btn variant-soft-surface" onclick={closeDeleteModal}>取消</button>
					<button type="button" class="btn variant-filled-primary" onclick={confirmDeleteGoal} disabled={isMutating}>确认删除</button>
				</div>
			</div>
		</div>
	{/if}
</div>

<style>
	.goals-page {
		max-width: 1120px;
		margin: 0 auto;
		padding: 12px 0 40px;
		display: grid;
		gap: 20px;
	}

	.goals-content {
		position: relative;
		display: grid;
		gap: 20px;
	}

	.page-refreshing {
		opacity: 0.92;
	}

	.refresh-indicator {
		position: sticky;
		top: 8px;
		z-index: 2;
		justify-self: end;
		padding: 8px 12px;
		border-radius: 999px;
		background: rgba(255, 248, 240, 0.96);
		border: 1px solid rgba(140, 90, 60, 0.18);
		color: var(--color-ink-soft);
		font-size: 0.88rem;
		box-shadow: var(--shadow-card);
	}

	.loading-state,
	.empty-goals {
		padding: 40px 28px;
		text-align: center;
	}

	.goals-hero,
	.checkin-panel,
	.create-panel,
	.history-panel,
	.detail-panel,
	.goal-progress-card,
	.modal-card {
		background:
			linear-gradient(180deg, rgba(255, 252, 247, 0.98) 0%, rgba(246, 238, 227, 0.96) 100%),
			var(--color-panel);
	}

	.goals-hero,
	.create-panel,
	.section-head,
	.goal-progress-head,
	.detail-head,
	.history-head,
	.modal-head,
	.modal-actions {
		display: flex;
		justify-content: space-between;
		gap: 16px;
		align-items: flex-start;
	}

	.goals-hero,
	.checkin-panel,
	.create-panel,
	.history-panel,
	.detail-panel,
	.goal-progress-card {
		padding: 24px;
	}

	.goals-hero {
		background:
			linear-gradient(135deg, rgba(255, 253, 249, 0.99) 0%, rgba(249, 243, 236, 0.96) 48%, rgba(239, 227, 210, 0.92) 100%),
			var(--color-panel);
		min-height: 152px;
		box-shadow: 0 20px 42px rgba(83, 58, 35, 0.08);
	}

	.checkin-panel {
		background:
			linear-gradient(180deg, rgba(255, 252, 247, 0.99) 0%, rgba(249, 241, 231, 0.96) 100%),
			var(--color-panel);
		box-shadow: 0 22px 44px rgba(83, 58, 35, 0.09);
	}

	.goals-kicker,
	.section-kicker {
		margin: 0 0 8px;
		font-size: 12px;
		letter-spacing: 0.16em;
		text-transform: uppercase;
		color: var(--color-muted);
	}

	.goals-hero h1,
	.checkin-panel h2,
	.detail-panel h2,
	.create-panel h2,
	.history-panel h2,
	.empty-goals h2 {
		margin: 0;
		font-family: var(--font-family-display);
		font-weight: 600;
	}

	.goals-hero h1 {
		font-size: 42px;
		letter-spacing: -0.03em;
	}

	.goals-copy,
	.section-copy,
	.goal-description {
		margin: 12px 0 0;
		line-height: 1.72;
		color: var(--color-ink-soft);
	}

	.hero-actions {
		display: flex;
		flex-direction: column;
		align-items: flex-end;
		gap: 12px;
		justify-content: flex-end;
	}

	.hero-date-chip,
	.checkin-summary-chip {
		display: inline-flex;
		align-items: center;
		gap: 10px;
		padding: 10px 14px;
		border: 1px solid rgba(140, 90, 60, 0.18);
		border-radius: 999px;
		background: rgba(255, 248, 240, 0.86);
		color: var(--color-ink-soft);
		box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.7);
	}

	.hero-date-chip span,
	.checkin-summary-chip span,
	.overview-stat span,
	.overview-stat p,
	.period-detail-item small,
	.period-day-head span {
		color: var(--color-muted);
	}

	.hero-date-chip strong,
	.checkin-summary-chip strong {
		font-family: var(--font-family-mono);
		font-size: 0.92rem;
		color: var(--color-ink);
	}

	.date-stepper,
	.goal-card-actions,
	.checkin-quantity,
	.calendar-nav {
		display: flex;
		gap: 10px;
		align-items: center;
		flex-wrap: wrap;
	}

	.section-head {
		gap: 20px;
		padding-bottom: 18px;
		border-bottom: 1px solid rgba(113, 91, 70, 0.08);
	}

	.checkin-list,
	.period-detail-list {
		display: grid;
		gap: 14px;
		margin-top: 18px;
	}

	.checkin-list {
		grid-template-columns: repeat(2, minmax(0, 1fr));
	}

	.checkin-actions {
		margin-top: 22px;
		display: flex;
		justify-content: flex-end;
		padding-top: 18px;
		border-top: 1px solid rgba(113, 91, 70, 0.08);
	}

	.checkin-row,
	.period-detail-item {
		display: flex;
		justify-content: space-between;
		gap: 16px;
		align-items: center;
	}

	.checkin-row {
		padding: 18px;
		border: 1px solid rgba(113, 91, 70, 0.12);
		border-radius: 20px;
		background: rgba(255, 252, 247, 0.82);
		box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.72);
	}

	.checkin-goal-name {
		margin: 0;
		font-size: 1.08rem;
		font-weight: 600;
	}

	.goal-description {
		font-size: 0.93rem;
	}

	.checkin-tags,
	.goal-card-footer {
		display: flex;
		gap: 8px;
		margin-top: 8px;
		flex-wrap: wrap;
	}

	.goal-inline-pill {
		display: inline-flex;
		align-items: center;
		padding: 6px 10px;
		border-radius: 999px;
		background: rgba(140, 90, 60, 0.09);
		color: var(--color-ink-soft);
		font-size: 0.82rem;
	}

	.subtle-pill {
		background: rgba(95, 76, 58, 0.07);
	}

	.quantity-input {
		width: 9rem;
		text-align: right;
		font-weight: 600;
	}

	.checkin-toggle {
		display: inline-flex;
		align-items: center;
		gap: 14px;
		padding: 12px 16px;
		border-radius: 18px;
		border: 1px solid var(--color-border);
		background: rgba(251, 246, 238, 0.92);
		cursor: pointer;
		transition: background 0.2s ease, transform 0.2s ease;
	}

	.checkin-toggle:hover:not(:disabled) {
		transform: translateY(-1px);
	}

	.checkin-done,
	.goal-card-strong {
		border-color: rgba(140, 90, 60, 0.3);
		box-shadow: 0 16px 28px rgba(83, 58, 35, 0.08);
	}

	.checkin-row:focus-within {
		border-color: rgba(140, 90, 60, 0.28);
		box-shadow: 0 16px 28px rgba(83, 58, 35, 0.08);
	}

	.checkin-done {
		background: linear-gradient(180deg, rgba(242, 229, 213, 0.92) 0%, rgba(235, 220, 201, 0.9) 100%);
	}

	.overview-strip {
		display: grid;
		grid-template-columns: repeat(3, minmax(0, 1fr));
		gap: 16px;
	}

	.overview-stat {
		padding: 18px 20px;
		display: grid;
		gap: 8px;
		background: linear-gradient(180deg, rgba(255, 252, 247, 0.98) 0%, rgba(247, 241, 232, 0.86) 100%);
		box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.72);
	}

	.overview-stat p,
	.overview-stat span {
		margin: 0;
	}

	.overview-stat strong {
		font-size: 2rem;
		font-weight: 700;
		letter-spacing: -0.03em;
	}

	.stats-panel,
	.goal-progress-grid,
	.detail-grid,
	.history-stats {
		display: grid;
		gap: 18px;
	}

	.goal-progress-grid,
	.detail-grid {
		grid-template-columns: repeat(2, minmax(0, 1fr));
	}

	.goal-progress-card {
		padding: 22px;
		transition: transform 0.2s ease, box-shadow 0.2s ease;
	}

	.goal-progress-card:hover {
		transform: translateY(-1px);
	}

	.goal-progress-head h2,
	.period-day-head strong {
		margin: 0;
	}

	.goal-progress-bar {
		position: relative;
		height: 12px;
		margin-top: 18px;
		border-radius: 999px;
		background: #efe4d6;
		overflow: hidden;
	}

	.goal-progress-fill {
		display: block;
		height: 100%;
		border-radius: inherit;
		background: linear-gradient(90deg, #9f6645 0%, #c38a59 100%);
	}

	.goal-progress-meta {
		display: flex;
		justify-content: space-between;
		gap: 16px;
		margin-top: 14px;
		font-size: 0.94rem;
		color: var(--color-ink-soft);
	}

	.period-day-block {
		display: grid;
		gap: 10px;
		padding: 16px 18px;
		border: 1px solid rgba(113, 91, 70, 0.11);
		border-radius: 18px;
		background: rgba(255, 252, 247, 0.78);
		box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.72);
	}

	.period-day-head {
		display: flex;
		justify-content: space-between;
		gap: 12px;
		align-items: center;
	}

	.create-panel {
		align-items: center;
		background:
			linear-gradient(180deg, rgba(255, 252, 247, 0.98) 0%, rgba(247, 241, 232, 0.9) 100%),
			var(--color-panel);
	}

	.modal-backdrop {
		position: fixed;
		inset: 0;
		display: grid;
		place-items: center;
		padding: 20px;
		background: rgba(23, 18, 14, 0.34);
		z-index: 50;
	}

	.modal-card {
		width: min(100%, 560px);
		max-height: calc(100vh - 40px);
		padding: 24px;
		overflow-y: auto;
		overscroll-behavior: contain;
	}

	.goal-form {
		display: grid;
		gap: 16px;
		margin-top: 18px;
	}

	.textarea-input {
		min-height: 96px;
		resize: vertical;
	}

	.delete-goal-name {
		margin: 16px 0 10px;
		color: var(--color-ink-soft);
	}

	@media (max-width: 960px) {
		.overview-strip,
		.goal-progress-grid,
		.checkin-list,
		.detail-grid {
			grid-template-columns: 1fr;
		}

		.hero-actions {
			align-items: flex-start;
		}

		.goals-hero {
			min-height: auto;
		}
	}

	@media (max-width: 720px) {
		.goals-page {
			padding-top: 8px;
			gap: 14px;
		}

		.goals-hero,
		.section-head,
		.create-panel,
		.goal-progress-head,
		.goal-progress-meta,
		.modal-head,
		.modal-actions,
		.history-head,
		.checkin-row,
		.period-day-head {
			flex-direction: column;
			align-items: stretch;
		}

		.goal-card-footer,
		.checkin-quantity,
		.date-stepper,
		.calendar-nav {
			align-items: stretch;
		}

		.checkin-actions {
			justify-content: stretch;
		}

		.goals-hero,
		.checkin-panel,
		.create-panel,
		.history-panel,
		.detail-panel,
		.goal-progress-card {
			padding: 18px;
		}

		.goals-hero h1 {
			font-size: 34px;
		}

		.checkin-row {
			padding: 16px;
		}

		.quantity-input {
			width: 100%;
			text-align: left;
		}
	}
</style>
