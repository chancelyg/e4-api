<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/stores.svelte';
	import {
		goalsAPI,
		type GoalCalendarDay,
		type GoalDashboard,
		type GoalDashboardItem,
		type GoalRange,
		type GoalType
	} from '$lib/api';
	import { formatLongDate, formatMonthLabel, formatWeekday, getTodayDateString, getYesterdayDateString, shiftMonth } from '$lib/date';

	type GoalFormState = {
		reactivateId: string;
		name: string;
		goalType: GoalType;
		unit: string;
		annualTarget: string | number;
		weeklyTarget: string | number;
	};

	type CalendarCell =
		| { kind: 'blank'; key: string }
		| { kind: 'day'; key: string; day: GoalCalendarDay };

	const rangeOptions: { value: GoalRange; label: string }[] = [
		{ value: 'week', label: '周' },
		{ value: 'month', label: '月' },
		{ value: 'quarter', label: '季' },
		{ value: 'year', label: '年' },
		{ value: 'all', label: '全部' }
	];

	let dashboard = $state<GoalDashboard | null>(null);
	let isLoading = $state(true);
	let isRefreshing = $state(false);
	let isSaving = $state(false);
	let isMutating = $state(false);
	let error = $state('');
	let selectedRange = $state<GoalRange>('month');
	let anchorDate = $state(getTodayDateString());
	let checkinDate = $state(getYesterdayDateString());
	let calendarMonth = $state(anchorDate.slice(0, 7));
	let selectedDate = $state(getYesterdayDateString());
	let showFormModal = $state(false);
	let isEditing = $state(false);
	let editingGoalId = $state<number | null>(null);
	let formError = $state('');
	let quantityDrafts = $state<Record<number, string>>({});
	let goalForm = $state<GoalFormState>(emptyGoalForm());

	onMount(() => {
		void initPage();
	});

	$effect(() => {
		if (!showFormModal) {
			return;
		}

		const handleKeydown = (event: KeyboardEvent) => {
			if (event.key === 'Escape') {
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
				range: selectedRange,
				date: anchorDate,
				checkin_date: checkinDate,
				month: calendarMonth
			});
			dashboard = result;
			checkinDate = result.checkin_date;
			calendarMonth = result.calendar_month;
			if (!findDayDetail(selectedDate)) {
				selectedDate = result.checkin_date;
			}
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

	function initializeQuantityDrafts(goals: GoalDashboardItem[]) {
		const nextDrafts: Record<number, string> = {};
		for (const goal of goals) {
			if (goal.goal_type === 'quantity') {
				nextDrafts[goal.id] = goal.checkin_record?.quantity?.toString() ?? '';
			}
		}
		quantityDrafts = nextDrafts;
	}

	function emptyGoalForm(): GoalFormState {
		return {
			reactivateId: '',
			name: '',
			goalType: 'checkbox',
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
			goalType: goal.goal_type,
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

	async function handleGoalSubmit(event: Event) {
		event.preventDefault();
		formError = '';
		isSaving = true;

		try {
			const reactivateId = goalForm.reactivateId ? Number(goalForm.reactivateId) : undefined;
			const reactivatedGoal = reactivateId ? dashboard?.inactive_goals.find((goal) => goal.id === reactivateId) ?? null : null;
			const annualTarget = parseOptionalNumber(goalForm.annualTarget);
			const payload = {
				reactivate_id: reactivateId,
				name: (reactivatedGoal?.name ?? goalForm.name).trim(),
				goal_type: reactivatedGoal?.goal_type ?? goalForm.goalType,
				unit: (reactivatedGoal?.goal_type ?? goalForm.goalType) === 'quantity' ? (reactivatedGoal?.unit ?? goalForm.unit).trim() : undefined,
				annual_target: isEditing ? (annualTarget ?? 0) : annualTarget,
				weekly_target: (reactivatedGoal?.goal_type ?? goalForm.goalType) === 'frequency' ? parseOptionalInt(goalForm.weeklyTarget) : undefined
			};

			if (isEditing && editingGoalId !== null) {
				await goalsAPI.update(editingGoalId, payload);
			} else {
				await goalsAPI.create(payload);
			}

			closeFormModal();
			await loadDashboard({ silent: true });
		} catch (err) {
			formError = err instanceof Error ? err.message : '保存目标失败';
		} finally {
			isSaving = false;
		}
	}

	async function handleGoalDelete(goal: GoalDashboardItem) {
		if (!confirm(`停用目标“${goal.name}”？历史记录会保留。`)) {
			return;
		}

		isMutating = true;
		try {
			await goalsAPI.delete(goal.id);
			await loadDashboard({ silent: true });
		} catch (err) {
			error = err instanceof Error ? err.message : '停用目标失败';
		} finally {
			isMutating = false;
		}
	}

	async function toggleCheckin(goal: GoalDashboardItem) {
		isMutating = true;
		try {
			if (goal.checkin_record) {
				await goalsAPI.deleteRecord(goal.id, checkinDate);
			} else {
				await goalsAPI.upsertRecord(goal.id, checkinDate);
			}
			await loadDashboard({ silent: true });
		} catch (err) {
			error = err instanceof Error ? err.message : '更新打卡失败';
		} finally {
			isMutating = false;
		}
	}

	async function saveQuantity(goal: GoalDashboardItem) {
		const quantity = Number(quantityDrafts[goal.id]);
		if (!Number.isFinite(quantity) || quantity <= 0) {
			error = '请输入大于 0 的数量';
			return;
		}

		isMutating = true;
		try {
			await goalsAPI.upsertRecord(goal.id, checkinDate, { quantity });
			await loadDashboard({ silent: true });
		} catch (err) {
			error = err instanceof Error ? err.message : '保存数量失败';
		} finally {
			isMutating = false;
		}
	}

	async function clearQuantity(goal: GoalDashboardItem) {
		if (!goal.checkin_record) return;

		isMutating = true;
		try {
			await goalsAPI.deleteRecord(goal.id, checkinDate);
			await loadDashboard({ silent: true });
		} catch (err) {
			error = err instanceof Error ? err.message : '清除打卡失败';
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
			goalType: inactiveGoal.goal_type,
			unit: inactiveGoal.unit,
			annualTarget: inactiveGoal.annual_target?.toString() ?? '',
			weeklyTarget: inactiveGoal.weekly_target?.toString() ?? ''
		};
	}

	function findDayDetail(date: string) {
		return dashboard?.day_details.find((item) => item.date === date) ?? null;
	}

	function selectDate(date: string) {
		selectedDate = date;
	}

	function changeRange(range: GoalRange) {
		selectedRange = range;
		void loadDashboard({ silent: true });
	}

	function switchCheckinDate(nextDate: string) {
		checkinDate = nextDate;
		selectedDate = nextDate;
		void loadDashboard({ silent: true });
	}

	function previousMonth() {
		calendarMonth = shiftMonth(calendarMonth, -1);
		void loadDashboard({ silent: true });
	}

	function nextMonth() {
		calendarMonth = shiftMonth(calendarMonth, 1);
		void loadDashboard({ silent: true });
	}

	function getProgressStyle(percent: number | null) {
		return `width: ${Math.max(0, Math.min(percent ?? 0, 100))}%`;
	}

	function getIntensityClass(level: number) {
		return `level-${level}`;
	}

	function getRangeLabel(range: GoalRange) {
		return rangeOptions.find((item) => item.value === range)?.label ?? '月';
	}

	function formatRangeStat(goal: GoalDashboardItem) {
		if (goal.goal_type === 'quantity') {
			return `${goal.range_quantity_total}${goal.unit || ''}`;
		}
		return `${goal.range_completed_count} 次`;
	}

	function formatAnnualSummary(goal: GoalDashboardItem) {
		if (goal.goal_type === 'quantity') {
			if (goal.annual_target !== null) {
				return `${goal.annual_quantity_total}/${goal.annual_target}${goal.unit}`;
			}
			return `本年 ${goal.annual_quantity_total}${goal.unit}`;
		}

		if (goal.goal_type === 'frequency') {
			if (goal.annual_target !== null) {
				return `${goal.annual_completed_count}/${goal.annual_target} 次`;
			}
			return `本年 ${goal.annual_completed_count} 次`;
		}

		return `本年 ${goal.annual_completed_count} 次`;
	}

	function getCheckinCompletedCount() {
		return dashboard?.goals.filter((goal) => goal.checkin_record).length ?? 0;
	}

	function getAnnualTargetCount() {
		return dashboard?.goals.filter((goal) => goal.annual_target !== null).length ?? 0;
	}

	function getRangeHighlightCount() {
		return dashboard?.goals.filter((goal) => goal.range_completed_count > 0 || goal.range_quantity_total > 0).length ?? 0;
	}

	function buildCalendarCells(days: GoalCalendarDay[] | undefined): CalendarCell[] {
		if (!days || days.length === 0) {
			return [];
		}

		const firstDay = new Date(`${days[0].date}T00:00:00`);
		const leadingBlanks = (firstDay.getDay() + 6) % 7;
		const cells: CalendarCell[] = [];

		for (let index = 0; index < leadingBlanks; index += 1) {
			cells.push({ kind: 'blank', key: `blank-${index}` });
		}

		for (const day of days) {
			cells.push({ kind: 'day', key: day.date, day });
		}

		return cells;
	}

	const selectedDayDetail = $derived(findDayDetail(selectedDate));
	const hasGoals = $derived((dashboard?.goals.length ?? 0) > 0);
	const calendarCells = $derived(buildCalendarCells(dashboard?.calendar_days));
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
				<h1>目标打卡</h1>
				<p class="goals-copy">把年度目标、昨日回顾和阶段表现放在一个页面里，快速看到差距与节奏。</p>
			</div>
			<div class="hero-actions">
				<div class="hero-date-chip">
					<span>锚点日期</span>
					<strong>{anchorDate}</strong>
				</div>
				<button class="btn variant-filled-primary" type="button" onclick={openCreateModal}>新增目标</button>
			</div>
		</section>

		{#if !hasGoals}
			<section class="empty-goals card">
				<h2>还没有目标</h2>
				<p>先创建几个想长期坚持的目标，再开始昨天/今天的回顾打卡。</p>
				<button class="btn variant-filled-primary" type="button" onclick={openCreateModal}>创建第一个目标</button>
			</section>
		{:else}
			<section class="overview-strip">
				<article class="overview-stat card">
					<p>活跃目标</p>
					<strong>{dashboard.goals.length}</strong>
					<span>维持一个小而稳定的年度清单</span>
				</article>
				<article class="overview-stat card">
					<p>{checkinDate === getYesterdayDateString() ? '昨日已记录' : '今日已记录'}</p>
					<strong>{getCheckinCompletedCount()}</strong>
					<span>共 {dashboard.goals.length} 项目标</span>
				</article>
				<article class="overview-stat card">
					<p>年度目标</p>
					<strong>{getAnnualTargetCount()}</strong>
					<span>显示年度进度条与剩余差距</span>
				</article>
				<article class="overview-stat card">
					<p>{getRangeLabel(selectedRange)}度活跃</p>
					<strong>{getRangeHighlightCount()}</strong>
					<span>{dashboard.range_start_date} - {dashboard.range_end_date}</span>
				</article>
			</section>

			<section class="goal-progress-grid">
				{#each dashboard.goals as goal (goal.id)}
					<article class:goal-card-strong={!!goal.checkin_record} class="goal-progress-card card">
						<div class="goal-progress-head">
							<div>
								<p class="goal-type-label">{goal.goal_type === 'checkbox' ? '坚持型' : goal.goal_type === 'quantity' ? '数值累计' : '频率型'}</p>
								<h2>{goal.name}</h2>
							</div>
							<div class="goal-card-actions">
								<button class="btn btn-sm variant-soft-surface" type="button" onclick={() => openEditModal(goal)}>编辑</button>
								<button class="btn btn-sm btn-quiet" type="button" onclick={() => handleGoalDelete(goal)} disabled={isMutating}>停用</button>
							</div>
						</div>

						<div class="goal-progress-bar">
							<span class="goal-progress-fill" style={getProgressStyle(goal.annual_progress_percent ?? goal.current_week_progress_percent ?? 0)}></span>
						</div>

						<div class="goal-progress-meta">
							<strong>{formatAnnualSummary(goal)}</strong>
							{#if goal.annual_remaining_value !== null}
								<span>还差 {goal.annual_remaining_value}{goal.goal_type === 'quantity' ? goal.unit : ' 次'}</span>
							{:else if goal.current_week_progress_percent !== null && goal.weekly_target !== null}
								<span>本周 {goal.current_week_completed_count}/{goal.weekly_target} · {goal.current_week_progress_percent}%</span>
							{:else}
								<span>{selectedRange === 'all' ? '全部范围' : `本${getRangeLabel(selectedRange)}`} {formatRangeStat(goal)}</span>
							{/if}
						</div>

						<div class="goal-card-footer">
							<p class="goal-range-note">当前范围：{formatRangeStat(goal)}</p>
							{#if goal.goal_type === 'frequency' && goal.weekly_target !== null}
								<span class="goal-inline-pill">周目标 {goal.weekly_target} 次</span>
							{:else if goal.goal_type === 'quantity' && goal.unit}
								<span class="goal-inline-pill">单位：{goal.unit}</span>
							{/if}
						</div>
					</article>
				{/each}
			</section>

			<section class="goals-controls card">
				<div>
					<p class="section-kicker">Check-in</p>
					<h2>{formatLongDate(checkinDate)}</h2>
					<p class="section-copy">只允许记录今天或昨天；这里默认展示昨天，适合早上回顾。完成的记录会直接停留在当前视图里，不再整页闪动。</p>
				</div>
				<div class="checkin-switches">
					<button class:active-chip={checkinDate === getYesterdayDateString()} class="btn variant-soft-surface" type="button" onclick={() => switchCheckinDate(getYesterdayDateString())}>昨天</button>
					<button class:active-chip={checkinDate === getTodayDateString()} class="btn variant-soft-surface" type="button" onclick={() => switchCheckinDate(getTodayDateString())}>今天</button>
					<div class="checkin-summary-chip">
						<strong>{getCheckinCompletedCount()}</strong>
						<span>/ {dashboard.goals.length} 已记录</span>
					</div>
				</div>
			</section>

			<section class="checkin-list card">
				{#each dashboard.goals as goal (goal.id)}
					<div class="checkin-row">
						<div>
							<p class="checkin-goal-name">{goal.name}</p>
							<div class="checkin-tags">
								<span class="goal-inline-pill">{goal.goal_type === 'checkbox' ? '坚持型' : goal.goal_type === 'quantity' ? '数值累计' : '频率型'}</span>
								{#if goal.annual_target !== null}
									<span class="goal-inline-pill subtle-pill">年度目标 {goal.annual_target}{goal.goal_type === 'quantity' ? goal.unit : ''}</span>
								{/if}
							</div>
							<p class="checkin-goal-meta">
								{#if goal.goal_type === 'quantity'}
									本{getRangeLabel(selectedRange)} {goal.range_quantity_total}{goal.unit}
								{:else}
									本{getRangeLabel(selectedRange)} {goal.range_completed_count} 次
								{/if}
							</p>
						</div>

						{#if goal.goal_type === 'quantity'}
							<div class="checkin-quantity">
								<input class="input quantity-input" type="number" min="0" step="0.1" bind:value={quantityDrafts[goal.id]} placeholder={`输入${goal.unit}`} />
								<button class="btn variant-filled-primary" type="button" onclick={() => saveQuantity(goal)} disabled={isMutating}>保存</button>
								{#if goal.checkin_record}
									<button class="btn variant-soft-surface" type="button" onclick={() => clearQuantity(goal)} disabled={isMutating}>清除</button>
								{/if}
								{#if goal.checkin_record?.quantity !== null && goal.checkin_record?.quantity !== undefined}
									<span class="goal-inline-pill">已记 {goal.checkin_record.quantity}{goal.unit}</span>
								{/if}
							</div>
						{:else}
							<button class:checkin-done={!!goal.checkin_record} class="checkin-toggle" type="button" onclick={() => toggleCheckin(goal)} disabled={isMutating}>
								<span>{goal.checkin_record ? '已完成' : '待记录'}</span>
								<strong>{goal.goal_type === 'frequency' && goal.weekly_target !== null ? `${goal.current_week_completed_count}/${goal.weekly_target}` : goal.checkin_record ? '✓' : '+'}</strong>
							</button>
						{/if}
					</div>
				{/each}
			</section>

			<section class="range-toolbar card">
				<div>
					<p class="section-kicker">Range</p>
					<h2>{dashboard.range_start_date} - {dashboard.range_end_date}</h2>
					<p class="section-copy">切换周、月、季、年与全部视图，重新查看每个目标的阶段表现。</p>
				</div>
				<div class="range-tabs">
					{#each rangeOptions as option}
						<button class:active-range={selectedRange === option.value} class="btn variant-soft-surface" type="button" onclick={() => changeRange(option.value)}>
							{option.label}
						</button>
					{/each}
				</div>
			</section>

			<section class="calendar-layout">
				<div class="calendar-panel card">
					<div class="calendar-head">
						<div>
							<p class="section-kicker">Calendar</p>
							<h2>{formatMonthLabel(calendarMonth)}</h2>
							<p class="section-copy">色块越深，代表那一天完成的目标越多。点某一天查看完整明细，右侧详情不会重新抖动。</p>
						</div>
						<div class="calendar-nav">
							<button class="btn variant-soft-surface" type="button" onclick={previousMonth}>上个月</button>
							<button class="btn variant-soft-surface" type="button" onclick={nextMonth}>下个月</button>
						</div>
					</div>

					<div class="calendar-weekdays">
						<span>一</span>
						<span>二</span>
						<span>三</span>
						<span>四</span>
						<span>五</span>
						<span>六</span>
						<span>日</span>
					</div>

					<div class="calendar-grid">
						{#each calendarCells as cell (cell.key)}
							{#if cell.kind === 'blank'}
								<div class="calendar-blank" aria-hidden="true"></div>
							{:else}
								<button class:calendar-day-active={selectedDate === cell.day.date} class={`calendar-day ${getIntensityClass(cell.day.intensity)}`} type="button" onclick={() => selectDate(cell.day.date)}>
									<span class="calendar-day-number">{Number(cell.day.date.slice(8, 10))}</span>
									<small>{cell.day.completed_goals}/{cell.day.total_goals}</small>
								</button>
							{/if}
						{/each}
					</div>

					<div class="calendar-legend">
						<span><i class="legend-swatch level-0"></i>空白</span>
						<span><i class="legend-swatch level-2"></i>部分完成</span>
						<span><i class="legend-swatch level-4"></i>高完成度</span>
					</div>
				</div>

				<aside class="calendar-detail card">
					{#if selectedDayDetail}
						<p class="section-kicker">Day Detail</p>
						<h2>{formatLongDate(selectedDayDetail.date)}</h2>
						<p class="calendar-detail-summary">完成 {selectedDayDetail.completed_goals} / {selectedDayDetail.total_goals} 项 · {formatWeekday(selectedDayDetail.date)}</p>
						<p class="calendar-detail-hint">左侧月历用于选择日期，右侧固定展示这一天的完整快照，便于安静回顾。</p>

						<div class="calendar-detail-list">
							{#each selectedDayDetail.items as item (item.goal_id)}
								<div class:detail-done={item.is_completed} class="calendar-detail-item">
									<div>
										<strong>{item.name}</strong>
										<p>{item.goal_type === 'quantity' ? (item.quantity !== null ? `${item.quantity}${item.unit}` : '未记录') : item.is_completed ? '已完成' : '未完成'}</p>
									</div>
									<span>{item.is_completed ? '✓' : '·'}</span>
								</div>
							{/each}
						</div>
					{:else}
						<p class="text-surface-500">选择某一天查看详细完成情况。</p>
					{/if}
				</aside>
			</section>
		{/if}
		</div>
	{/if}

	{#if showFormModal}
		<div
			class="modal-backdrop"
			role="presentation"
			onclick={(event) => event.target === event.currentTarget && closeFormModal()}
		>
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
							<span>重新启用已停用目标（可选）</span>
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
						<span>目标类型</span>
						<select class="input" bind:value={goalForm.goalType} disabled={isEditing || (!!goalForm.reactivateId && !isEditing)}>
							<option value="checkbox">坚持型</option>
							<option value="quantity">数值累计型</option>
							<option value="frequency">频率型</option>
						</select>
					</label>

					{#if goalForm.goalType === 'quantity'}
						<label class="label">
							<span>单位</span>
							<input class="input" type="text" bind:value={goalForm.unit} placeholder="km / 本 / 页" required disabled={!!goalForm.reactivateId && !isEditing} />
						</label>
					{/if}

					<label class="label">
						<span>年度目标（可选）</span>
						<input class="input" type="number" min="0" step="0.1" bind:value={goalForm.annualTarget} placeholder="例如 500" />
					</label>

					{#if goalForm.goalType === 'frequency'}
						<label class="label">
							<span>每周目标次数</span>
							<input class="input" type="number" min="1" step="1" bind:value={goalForm.weeklyTarget} placeholder="例如 3" required />
						</label>
					{/if}

					<div class="modal-actions">
						<button type="button" class="btn variant-soft-surface" onclick={closeFormModal}>取消</button>
						<button type="submit" class="btn variant-filled-primary" disabled={isSaving}>{isSaving ? '保存中...' : '保存目标'}</button>
					</div>
				</form>
			</div>
		</div>
	{/if}
</div>

<style>
	.goals-page {
		max-width: 1120px;
		margin: 0 auto;
		padding: 20px 0 40px;
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

	.hero-actions {
		display: flex;
		flex-direction: column;
		align-items: flex-end;
		gap: 12px;
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
	}

	.hero-date-chip span,
	.checkin-summary-chip span {
		font-size: 0.88rem;
		color: var(--color-muted);
	}

	.hero-date-chip strong,
	.checkin-summary-chip strong {
		font-family: var(--font-family-mono);
		font-size: 0.92rem;
		color: var(--color-ink);
	}

	.loading-state,
	.empty-goals {
		padding: 40px 28px;
		text-align: center;
	}

	.goals-hero {
		display: flex;
		justify-content: space-between;
		gap: 24px;
		align-items: flex-start;
		padding: 28px;
	}

	.goals-kicker,
	.section-kicker,
	.goal-type-label {
		margin: 0 0 8px;
		font-size: 12px;
		letter-spacing: 0.16em;
		text-transform: uppercase;
		color: var(--color-muted);
	}

	.goals-hero h1,
	.goals-controls h2,
	.range-toolbar h2,
	.calendar-panel h2,
	.calendar-detail h2,
	.empty-goals h2 {
		margin: 0;
		font-family: var(--font-family-display);
		font-weight: 600;
	}

	.goals-hero h1 {
		font-size: 42px;
	}

	.goals-copy,
	.section-copy {
		margin: 12px 0 0;
		max-width: 42rem;
		line-height: 1.8;
		color: var(--color-ink-soft);
	}

	.overview-strip {
		display: grid;
		grid-template-columns: repeat(4, minmax(0, 1fr));
		gap: 16px;
	}

	.overview-stat {
		padding: 18px 20px;
		display: grid;
		gap: 8px;
		background: linear-gradient(180deg, rgba(255, 252, 247, 0.98) 0%, rgba(247, 241, 232, 0.86) 100%);
	}

	.overview-stat p,
	.overview-stat span {
		margin: 0;
		color: var(--color-muted);
	}

	.overview-stat strong {
		font-size: 2rem;
		font-weight: 700;
		letter-spacing: -0.03em;
	}

	.goal-progress-grid {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 18px;
	}

	.goal-progress-card,
	.goals-controls,
	.checkin-list,
	.range-toolbar,
	.calendar-panel,
	.calendar-detail {
		padding: 24px;
	}

	.goal-progress-card {
		background:
			linear-gradient(180deg, rgba(255, 252, 247, 0.98) 0%, rgba(246, 238, 227, 0.96) 100%),
			var(--color-panel);
	}

	.goal-card-strong {
		border-color: rgba(140, 90, 60, 0.24);
		box-shadow: 0 16px 28px rgba(83, 58, 35, 0.08);
	}

	.goal-progress-head,
	.goals-controls,
	.range-toolbar,
	.calendar-head,
	.modal-head,
	.modal-actions {
		display: flex;
		justify-content: space-between;
		gap: 16px;
		align-items: flex-start;
	}

	.goal-progress-head h2 {
		margin: 0;
		font-size: 1.3rem;
	}

	.goal-card-actions,
	.checkin-switches,
	.range-tabs,
	.calendar-nav,
	.checkin-quantity {
		display: flex;
		gap: 10px;
		align-items: center;
		flex-wrap: wrap;
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

	.goal-range-note,
	.checkin-goal-meta,
	.calendar-detail-summary,
	.calendar-detail-item p {
		margin: 10px 0 0;
		font-size: 0.92rem;
		color: var(--color-muted);
	}

	.checkin-list {
		display: grid;
		gap: 14px;
		background:
			linear-gradient(180deg, rgba(255, 252, 247, 0.96) 0%, rgba(251, 248, 243, 0.96) 100%),
			var(--color-panel);
	}

	.goal-card-footer {
		display: flex;
		justify-content: space-between;
		gap: 12px;
		align-items: center;
		margin-top: 12px;
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

	.checkin-row,
	.calendar-detail-item {
		display: flex;
		justify-content: space-between;
		gap: 16px;
		align-items: center;
		padding: 16px 0;
		border-top: 1px solid var(--color-border);
	}

	.checkin-row:first-child,
	.calendar-detail-item:first-child {
		padding-top: 0;
		border-top: none;
	}

	.checkin-goal-name {
		margin: 0;
		font-size: 1.02rem;
		font-weight: 600;
	}

	.checkin-tags {
		display: flex;
		gap: 8px;
		margin-top: 8px;
		flex-wrap: wrap;
	}

	.checkin-toggle {
		display: inline-flex;
		align-items: center;
		gap: 14px;
		padding: 12px 16px;
		border-radius: 18px;
		border: 1px solid var(--color-border);
		background: var(--color-panel-muted);
		cursor: pointer;
		transition: background 0.2s ease, transform 0.2s ease;
	}

	.checkin-toggle:hover:not(:disabled),
	.calendar-day:hover {
		transform: translateY(-1px);
	}

	.checkin-toggle strong {
		font-size: 1.1rem;
	}

	.checkin-done {
		background: rgba(140, 90, 60, 0.14);
		border-color: rgba(140, 90, 60, 0.3);
	}

	.quantity-input {
		width: 9rem;
	}

	.active-chip,
	.active-range {
		background: #ebdfcf;
		border-color: rgba(140, 90, 60, 0.3);
		color: var(--color-ink);
	}

	.calendar-layout {
		display: grid;
		grid-template-columns: minmax(0, 1.3fr) minmax(320px, 0.7fr);
		gap: 20px;
	}

	.calendar-weekdays,
	.calendar-grid {
		display: grid;
		grid-template-columns: repeat(7, minmax(0, 1fr));
		gap: 10px;
	}

	.calendar-blank {
		min-height: 86px;
	}

	.calendar-weekdays {
		margin-top: 18px;
		color: var(--color-muted);
		font-size: 0.86rem;
	}

	.calendar-day {
		display: grid;
		gap: 8px;
		align-content: space-between;
		min-height: 86px;
		padding: 12px;
		border-radius: 18px;
		border: 1px solid var(--color-border);
		background: #fbf8f3;
		cursor: pointer;
		text-align: left;
		transition: transform 0.2s ease, border-color 0.2s ease, background 0.2s ease;
	}

	.calendar-day-number {
		font-size: 1rem;
		font-weight: 700;
	}

	.calendar-day small {
		color: var(--color-muted);
	}

	.calendar-day-active {
		border-color: rgba(140, 90, 60, 0.42);
		box-shadow: inset 0 0 0 1px rgba(140, 90, 60, 0.22);
	}

	.level-0 { background: #fbf8f3; }
	.level-1 { background: #f2e7da; }
	.level-2 { background: #e8d4bb; }
	.level-3 { background: #ddb58f; }
	.level-4 { background: #cb8d59; color: white; }

	.calendar-legend {
		display: flex;
		gap: 16px;
		margin-top: 16px;
		color: var(--color-muted);
		font-size: 0.85rem;
		flex-wrap: wrap;
	}

	.calendar-legend span {
		display: inline-flex;
		align-items: center;
		gap: 8px;
	}

	.legend-swatch {
		display: inline-block;
		width: 14px;
		height: 14px;
		border-radius: 4px;
		border: 1px solid var(--color-border);
	}

	.calendar-detail-list {
		margin-top: 18px;
	}

	.calendar-detail-hint {
		margin: 10px 0 0;
		font-size: 0.9rem;
		line-height: 1.7;
		color: var(--color-muted);
	}

	.detail-done strong {
		color: var(--color-ink);
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
		background:
			linear-gradient(180deg, rgba(255, 252, 247, 0.99) 0%, rgba(247, 240, 231, 0.98) 100%),
			var(--color-panel);
	}

	.goal-form {
		display: grid;
		gap: 16px;
		margin-top: 18px;
	}

	.goal-form :global(select:disabled),
	.goal-form :global(input:disabled) {
		opacity: 0.72;
		cursor: not-allowed;
	}

	@media (max-width: 960px) {
		.overview-strip,
		.goal-progress-grid,
		.calendar-layout {
			grid-template-columns: 1fr;
		}

		.hero-actions {
			align-items: flex-start;
		}
	}

	@media (max-width: 720px) {
		.goals-hero,
		.goals-controls,
		.range-toolbar,
		.calendar-head,
		.modal-head,
		.modal-actions,
		.checkin-row {
			flex-direction: column;
			align-items: stretch;
		}

		.calendar-grid,
		.calendar-weekdays {
			gap: 8px;
		}

		.goal-progress-meta,
		.goal-card-footer {
			flex-direction: column;
			align-items: flex-start;
		}

		.calendar-day {
			min-height: 72px;
			padding: 10px;
		}

		.calendar-blank {
			min-height: 72px;
		}
	}
</style>
