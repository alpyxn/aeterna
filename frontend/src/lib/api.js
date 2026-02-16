export const API_BASE = import.meta.env.VITE_API_URL || "http://localhost:3000";

export async function apiRequest(path, options = {}) {
	const { headers, ...rest } = options;
	const base = API_BASE.endsWith("/") ? API_BASE.slice(0, -1) : API_BASE;
	const normalizedPath = path.startsWith("/") ? path : `/${path}`;
	const response = await fetch(`${base}${normalizedPath}`, {
		credentials: "include",
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

// Upload a file attachment to a message (multipart/form-data)
export async function uploadFile(messageId, file) {
	const formData = new FormData();
	formData.append("file", file);

	const base = API_BASE.endsWith("/") ? API_BASE.slice(0, -1) : API_BASE;
	const response = await fetch(`${base}/messages/${messageId}/attachments`, {
		method: "POST",
		credentials: "include",
		body: formData,
		// Do NOT set Content-Type — browser sets it with correct boundary
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
			(rawText ? rawText : `Upload failed (${response.status})`);
		throw new Error(message);
	}

	return data;
}

// Delete a file attachment
export async function deleteAttachment(messageId, attachmentId) {
	return apiRequest(`/messages/${messageId}/attachments/${attachmentId}`, {
		method: "DELETE",
	});
}

// List attachments for a message
export async function listAttachments(messageId) {
	return apiRequest(`/messages/${messageId}/attachments`);
}
