const WEEKDAYS = ['星期日', '星期一', '星期二', '星期三', '星期四', '星期五', '星期六'];
const MONTH_NAMES = ['一月', '二月', '三月', '四月', '五月', '六月', '七月', '八月', '九月', '十月', '十一月', '十二月'];

function pad2(value: number): string {
	return value.toString().padStart(2, '0');
}

function parseDateParts(dateStr: string): [number, number, number] {
	const [year, month, day] = dateStr.split('-').map(Number);
	return [year, month, day];
}

function createLocalDate(dateStr: string): Date {
	const [year, month, day] = parseDateParts(dateStr);
	return new Date(year, month - 1, day);
}

export function getTodayDateString(): string {
	const now = new Date();
	return `${now.getFullYear()}-${pad2(now.getMonth() + 1)}-${pad2(now.getDate())}`;
}

export function getMonthRange(monthStr: string): { startDate: string; endDate: string } {
	const [year, month] = monthStr.split('-').map(Number);
	const lastDay = new Date(year, month, 0).getDate();

	return {
		startDate: `${year}-${pad2(month)}-01`,
		endDate: `${year}-${pad2(month)}-${pad2(lastDay)}`
	};
}

export function formatMonthLabel(monthStr: string): string {
	if (!monthStr) return '';
	const [year, month] = monthStr.split('-').map(Number);
	return `${year}年${MONTH_NAMES[month - 1]}`;
}

export function formatWeekday(dateStr: string): string {
	return WEEKDAYS[createLocalDate(dateStr).getDay()];
}

export function formatLongDate(dateStr: string): string {
	const [year, month, day] = parseDateParts(dateStr);
	return `${year}年${month}月${day}日 ${formatWeekday(dateStr)}`;
}
