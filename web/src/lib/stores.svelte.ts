import { authAPI, type LoginResponse } from './api';

type LoginResult =
	| LoginResponse
	| {
			needs_2fa: true;
			challenge_token: string;
			success?: false;
			username?: string;
	  };

function createAuthStore() {
	let isLoggedIn = $state(false);
	let username = $state('');
	let isLoading = $state(true);
	let hasInitialized = $state(false);

	async function checkAuth() {
		isLoading = true;
		try {
			const status = await authAPI.status();
			isLoggedIn = status.is_logged_in;
			username = status.username || '';
		} catch {
			isLoggedIn = false;
			username = '';
		} finally {
			isLoading = false;
			hasInitialized = true;
		}
	}

	async function ensureInitialized() {
		if (hasInitialized) {
			return;
		}
		await checkAuth();
	}

	async function login(usernameInput: string, password: string): Promise<LoginResult> {
		const result = await authAPI.loginStep1({ username: usernameInput, password });
		if (result.needs_2fa && result.challenge_token) {
			return { needs_2fa: true, challenge_token: result.challenge_token };
		}
		if (result.success) {
			isLoggedIn = true;
			username = result.username;
			hasInitialized = true;
		}
		return result;
	}

	async function verify2FA(code: string, challengeToken: string): Promise<LoginResponse> {
		const result = await authAPI.loginStep2({ code, challenge_token: challengeToken });
		if (result.success) {
			isLoggedIn = true;
			username = result.username;
			hasInitialized = true;
		}
		return result;
	}

	async function logout() {
		await authAPI.logout();
		isLoggedIn = false;
		username = '';
		hasInitialized = true;
	}

	return {
		get isLoggedIn() {
			return isLoggedIn;
		},
		get username() {
			return username;
		},
		get isLoading() {
			return isLoading;
		},
		get hasInitialized() {
			return hasInitialized;
		},
		checkAuth,
		ensureInitialized,
		login,
		verify2FA,
		logout
	};
}

export type AuthStore = ReturnType<typeof createAuthStore>;

export const auth: AuthStore = createAuthStore();
