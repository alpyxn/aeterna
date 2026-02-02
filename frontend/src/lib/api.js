export const API_BASE = import.meta.env.VITE_API_URL || "http://localhost:3000";

export async function apiRequest(path, options = {}) {
	const { headers, ...rest } = options;
	const base = API_BASE.endsWith("/") ? API_BASE.slice(0, -1) : API_BASE;
	const normalizedPath = path.startsWith("/") ? path : `/${path}`;
	const response = await fetch(`${base}${normalizedPath}`, {
		...rest,
		headers: {
			"Content-Type": "application/json",
			...(headers || {}),
		},
	});

	let data = null;
	let rawText = "";
	try {
		rawText = await response.text();
		data = rawText ? JSON.parse(rawText) : null;
	} catch {
		data = null;
	}

	if (!response.ok) {
		const message =
			data?.error ||
			data?.message ||
			(rawText ? rawText : `Request failed (${response.status})`);
		throw new Error(message);
	}

	return data;
}
