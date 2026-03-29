const API_BASE = '/api';

interface LoginRequest {
	username: string;
	password: string;
}

interface LoginResponse {
	success: boolean;
	username: string;
	needs_2fa?: boolean;
	challenge_token?: string;
}

interface LoginStep1Request {
	username: string;
	password: string;
}

interface LoginStep2Request {
	code: string;
	challenge_token: string;
}

interface AuthStatus {
	is_logged_in: boolean;
	username?: string;
}

interface Diary {
	id: number;
	content: string;
	create_date: string;
	created_at: string;
	updated_at: string;
}

interface DiaryListResponse {
	diaries: Diary[];
	total: number;
}

interface DiaryStats {
	total_count: number;
	max_consecutive_days: number;
	start_date: string;
	end_date: string;
	time_span_days: number;
}

async function fetchAPI<T>(path: string, options?: RequestInit): Promise<T> {
	const response = await fetch(`${API_BASE}${path}`, {
		credentials: 'include',
		headers: {
			'Content-Type': 'application/json',
			...options?.headers
		},
		...options
	});

	if (!response.ok) {
		const error = await response.json().catch(() => ({ error: '请求失败' }));
		throw new Error(error.error || `HTTP ${response.status}`);
	}

	return response.json();
}

export const authAPI = {
	login: (data: LoginRequest) =>
		fetchAPI<LoginResponse>('/auth/login', {
			method: 'POST',
			body: JSON.stringify(data)
		}),

	loginStep1: (data: LoginStep1Request) =>
		fetchAPI<LoginResponse>('/auth/login-step1', {
			method: 'POST',
			body: JSON.stringify(data)
		}),

	loginStep2: (data: LoginStep2Request) =>
		fetchAPI<LoginResponse>('/auth/login-step2', {
			method: 'POST',
			body: JSON.stringify(data)
		}),

	logout: () =>
		fetchAPI<{ success: boolean }>('/auth/logout', {
			method: 'POST'
		}),

	status: () =>
		fetchAPI<AuthStatus>('/auth/status')
};

export const diaryAPI = {
	list: (params?: { page?: number; per_page?: number; search?: string; start_date?: string; end_date?: string }) => {
		const query = new URLSearchParams();
		const trimmedSearch = params?.search?.trim();
		if (params?.page) query.set('page', params.page.toString());
		if (params?.per_page) query.set('per_page', params.per_page.toString());
		if (trimmedSearch) query.set('search', trimmedSearch);
		if (params?.start_date) query.set('start_date', params.start_date);
		if (params?.end_date) query.set('end_date', params.end_date);

		return fetchAPI<DiaryListResponse>(`/diary?${query}`);
	},

	get: (id: number) =>
		fetchAPI<Diary>(`/diary/${id}`),

	create: (data: { content: string; create_date: string }) =>
		fetchAPI<Diary>('/diary', {
			method: 'POST',
			body: JSON.stringify(data)
		}),

	update: (id: number, data: { content?: string; create_date?: string }) =>
		fetchAPI<Diary>(`/diary/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data)
		}),

	delete: (id: number) =>
		fetchAPI<{ success: boolean }>(`/diary/${id}`, {
			method: 'DELETE'
		}),

	stats: () =>
		fetchAPI<DiaryStats>('/diary/stats')
};

export type { Diary, DiaryListResponse, DiaryStats, AuthStatus, LoginResponse };
