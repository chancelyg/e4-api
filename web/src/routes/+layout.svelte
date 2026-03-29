<script lang="ts">
	import '../app.css';
	import { auth } from '$lib/stores.svelte';
	import { onMount } from 'svelte';

	let { children } = $props();

	onMount(() => {
		auth.ensureInitialized();
	});

	async function handleLogout() {
		await auth.logout();
		window.location.href = '/';
	}
</script>

<div class="app-shell">
	{#if auth.isLoggedIn}
		<aside class="app-sidebar">
			<a class="sidebar-brand" href="/diary">
				<p class="sidebar-kicker">E4 Diary</p>
				<h1>记录与回看</h1>
				<p class="sidebar-meta">把日常整理成清晰、可检索的个人档案。</p>
			</a>

			<nav class="sidebar-nav">
				<a href="/diary" class="sidebar-link">日记列表</a>
				<a href="/diary/new" class="sidebar-link sidebar-link-secondary">写新日记</a>
			</nav>

			<div class="sidebar-footer">
				<div>
					<p class="sidebar-user-label">当前用户</p>
					<p class="sidebar-user-name">{auth.username}</p>
				</div>
				<button onclick={handleLogout} class="btn btn-quiet">退出</button>
			</div>
		</aside>
	{/if}

	<main class="app-content" class:app-content-wide={!auth.isLoggedIn}>
		{#if auth.isLoading && !auth.hasInitialized}
			<div class="page-loading">正在初始化...</div>
		{:else}
			{@render children()}
		{/if}
	</main>
</div>

<style>
	.app-shell {
		display: grid;
		grid-template-columns: 18rem minmax(0, 1fr);
		min-height: 100vh;
		max-width: 1440px;
		margin: 0 auto;
		padding: 24px;
		gap: 24px;
	}

	.app-sidebar {
		display: flex;
		flex-direction: column;
		justify-content: space-between;
		gap: 24px;
		padding: 24px;
		border: 1px solid var(--color-border);
		border-radius: 28px;
		background: var(--color-panel);
		box-shadow: var(--shadow-soft);
		position: sticky;
		top: 24px;
		height: calc(100vh - 48px);
	}

	.sidebar-brand {
		display: block;
		text-decoration: none;
		color: inherit;
	}

	.sidebar-kicker {
		margin: 0 0 10px;
		font-size: 12px;
		letter-spacing: 0.16em;
		text-transform: uppercase;
		color: var(--color-muted);
	}

	.sidebar-brand h1 {
		margin: 0;
		font-family: var(--font-family-display);
		font-size: 32px;
		font-weight: 600;
		line-height: 1.1;
	}

	.sidebar-meta {
		margin: 12px 0 0;
		font-size: 14px;
		line-height: 1.7;
		color: var(--color-muted);
	}

	.sidebar-nav {
		display: grid;
		gap: 12px;
	}

	.sidebar-link {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 14px 16px;
		border: 1px solid var(--color-border);
		border-radius: 16px;
		background: var(--color-panel-muted);
		text-decoration: none;
		font-size: 15px;
		font-weight: 600;
		transition: border-color 0.2s ease, background 0.2s ease, transform 0.2s ease;
	}

	.sidebar-link::after {
		content: '->';
		font-size: 13px;
		color: var(--color-muted);
	}

	.sidebar-link:hover {
		transform: translateY(-1px);
		border-color: var(--color-ink-soft);
		background: #f4f0e9;
	}

	.sidebar-link-secondary {
		background: #fcfaf6;
	}

	.sidebar-footer {
		display: flex;
		align-items: flex-end;
		justify-content: space-between;
		gap: 16px;
		padding-top: 18px;
		border-top: 1px solid var(--color-border);
	}

	.sidebar-user-label,
	.sidebar-user-name {
		margin: 0;
	}

	.sidebar-user-label {
		font-size: 12px;
		color: var(--color-muted);
		letter-spacing: 0.08em;
		text-transform: uppercase;
	}

	.sidebar-user-name {
		margin-top: 6px;
		font-size: 16px;
		font-weight: 600;
	}

	.app-content {
		min-width: 0;
	}

	.app-content-wide {
		grid-column: 1 / -1;
	}

	.page-loading {
		display: flex;
		align-items: center;
		justify-content: center;
		min-height: calc(100vh - 48px);
		color: var(--color-muted);
	}

	@media (max-width: 920px) {
		.app-shell {
			grid-template-columns: 1fr;
			padding: 16px;
			gap: 16px;
		}

		.app-sidebar {
			position: static;
			height: auto;
			padding: 20px;
		}

		.sidebar-footer {
			align-items: center;
		}
	}
</style>
