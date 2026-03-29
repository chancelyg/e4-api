<script lang="ts">
	import { goto } from '$app/navigation';
	import { diaryAPI } from '$lib/api';
	import { getTodayDateString } from '$lib/date';

	let content = $state('');
	let createDate = $state(getTodayDateString());
	let isSaving = $state(false);
	let error = $state('');

	async function handleSave() {
		if (!content.trim()) {
			error = '请输入日记内容';
			return;
		}

		isSaving = true;
		error = '';

		try {
			await diaryAPI.create({ content, create_date: createDate });
			goto('/diary');
		} catch (err) {
			error = err instanceof Error ? err.message : '保存失败';
		} finally {
			isSaving = false;
		}
	}

	function handleCancel() {
		goto('/diary');
	}
</script>

<div class="editor-page">
	<div class="editor-head">
		<div>
			<p class="editor-kicker">New Entry</p>
			<h2>写日记</h2>
		</div>
		<div class="editor-actions">
			<button onclick={handleCancel} class="btn variant-soft-surface">取消</button>
			<button onclick={handleSave} class="btn variant-filled-primary" disabled={isSaving}>
				{isSaving ? '保存中...' : '保存'}
			</button>
		</div>
	</div>

	{#if error}
		<div class="alert variant-soft-error">
			<p>{error}</p>
		</div>
	{/if}

	<section class="card editor-card">
		<label class="editor-date-field">
			<span>日期</span>
			<input type="date" bind:value={createDate} class="input editor-date-input" />
		</label>

		<textarea
			bind:value={content}
			placeholder="记录今天的心情、想法和经历..."
			class="editor-textarea"
		></textarea>
	</section>
</div>

<style>
	.editor-page {
		max-width: 920px;
		margin: 0 auto;
		padding: 20px 0 40px;
	}

	.editor-head {
		display: flex;
		justify-content: space-between;
		align-items: end;
		gap: 16px;
		margin-bottom: 16px;
	}

	.editor-kicker {
		margin: 0 0 8px;
		font-size: 12px;
		letter-spacing: 0.16em;
		text-transform: uppercase;
		color: var(--color-muted);
	}

	.editor-head h2 {
		margin: 0;
		font-family: var(--font-family-display);
		font-size: 2.2rem;
		font-weight: 600;
	}

	.editor-actions {
		display: flex;
		gap: 10px;
		flex-wrap: wrap;
	}

	.editor-card {
		padding: 22px;
	}

	.editor-date-field {
		display: inline-grid;
		gap: 8px;
		margin-bottom: 18px;
	}

	.editor-date-field span {
		font-size: 13px;
		color: var(--color-muted);
	}

	.editor-date-input {
		width: 220px;
	}

	@media (max-width: 640px) {
		.editor-page {
			padding: 8px 0 28px;
		}

		.editor-head {
			align-items: stretch;
			flex-direction: column;
		}

		.editor-actions :global(.btn),
		.editor-date-input {
			width: 100%;
		}

		.editor-card {
			padding: 16px;
		}
	}
</style>
