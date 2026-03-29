<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { diaryAPI, type DiaryStats } from '$lib/api';
	import { auth } from '$lib/stores.svelte';

	let stats = $state<DiaryStats | null>(null);
	let isLoading = $state(true);
	let error = $state('');

	onMount(() => {
		void initPage();
	});

	async function initPage() {
		await auth.ensureInitialized();
		if (!auth.isLoggedIn) {
			goto('/');
			return;
		}
		loadStats();
	}

	async function loadStats() {
		try {
			stats = await diaryAPI.stats();
		} catch (err) {
			error = err instanceof Error ? err.message : '加载统计失败';
		} finally {
			isLoading = false;
		}
	}
</script>

<div class="stats-page p-6 max-w-4xl mx-auto">
	<div class="stats-header mb-6">
		<p class="stats-kicker">OVERVIEW</p>
		<h2 class="text-2xl font-bold m-0">记录统计</h2>
	</div>

	{#if error}
		<div class="alert variant-soft-error mb-4">
			<p>{error}</p>
		</div>
	{/if}

	{#if isLoading}
		<div class="text-center py-12">
			<p class="text-surface-500">加载中...</p>
		</div>
	{:else if stats && stats.total_count > 0}
		<div class="stats-grid mb-8">
			<div class="stat-card">
				<div class="stat-value">{stats.total_count}</div>
				<div class="stat-label">总篇数</div>
			</div>
			<div class="stat-card">
				<div class="stat-value">{stats.max_consecutive_days}</div>
				<div class="stat-label">最大连续天数</div>
			</div>
			<div class="stat-card">
				<div class="stat-value">{stats.time_span_days}</div>
				<div class="stat-label">时间跨度(天)</div>
			</div>
			<div class="stat-card">
				<div class="stat-value">
					{stats.total_count > 0 && stats.time_span_days > 0 
						? (stats.total_count / stats.time_span_days * 100).toFixed(1) 
						: '0'}%
				</div>
				<div class="stat-label">记录频率</div>
			</div>
		</div>

		<div class="card p-6">
			<h3 class="text-lg font-bold mb-4">时间范围</h3>
			<div class="flex items-center gap-4">
				<div class="text-center flex-1">
					<p class="text-sm text-surface-500 mb-1">开始日期</p>
					<p class="font-mono text-lg">{stats.start_date}</p>
				</div>
				<div class="text-surface-400">→</div>
				<div class="text-center flex-1">
					<p class="text-sm text-surface-500 mb-1">结束日期</p>
					<p class="font-mono text-lg">{stats.end_date}</p>
				</div>
			</div>
		</div>
	{:else}
		<div class="text-center py-12">
			<p class="text-surface-500 mb-4">还没有日记数据</p>
			<a href="/diary/new" class="btn variant-filled-primary">
				写第一篇日记
			</a>
		</div>
	{/if}
</div>

<style>
	.stats-page {
		padding-top: 2rem;
	}

	.stats-header {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.stats-kicker {
		margin: 0;
		font-size: 12px;
		letter-spacing: 0.18em;
		color: var(--color-surface-500);
	}

	.stats-grid {
		display: grid;
		grid-template-columns: repeat(4, minmax(0, 1fr));
		gap: 1rem;
	}

	@media (max-width: 768px) {
		.stats-grid {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}
	}

	@media (max-width: 560px) {
		.stats-grid {
			grid-template-columns: 1fr;
		}
	}
</style>
