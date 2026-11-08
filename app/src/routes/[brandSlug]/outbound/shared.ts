// Helpers shared by the Outbound screens.

export function errorMessage(err: unknown, fallback: string): string {
	return err instanceof Error && err.message ? err.message : fallback;
}

// The API sends rates as percents (0 to 100).
export function percent(rate: number): string {
	return `${(rate || 0).toFixed(1)}%`;
}

// Which rows of a list a page shows: "1 to 50 of 120".
export function pageRange(offset: number, shown: number, total: number): string {
	if (shown === 0) return `0 of ${total}`;
	return `${offset + 1} to ${offset + shown} of ${total}`;
}

export const BOUNCE_LIMIT = 3;

export const CLASSIFICATIONS: { value: string; label: string }[] = [
	{ value: 'interested', label: 'Interested' },
	{ value: 'not_interested', label: 'Not interested' },
	{ value: 'question', label: 'Question' },
	{ value: 'out_of_office', label: 'Out of office' },
	{ value: 'wrong_person', label: 'Wrong person' },
	{ value: 'unsubscribe', label: 'Unsubscribe' },
	{ value: 'complaint', label: 'Complaint' }
];

export function classificationLabel(value: string): string {
	return CLASSIFICATIONS.find((c) => c.value === value)?.label || value || 'Not sorted';
}
