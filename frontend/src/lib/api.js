export const API_BASE = import.meta.env.VITE_API_URL || "/api";
const API_ROOT = API_BASE.endsWith("/") ? API_BASE.slice(0, -1) : API_BASE;

function buildApiUrl(path) {
	const normalizedPath = path.startsWith("/") ? path : `/${path}`;
	return `${API_ROOT}${normalizedPath}`;
}

async function parseResponse(response, errorPrefix = "Request failed") {
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
			(rawText ? rawText : `${errorPrefix} (${response.status})`);
		throw new Error(message);
	}

	return data;
}

export async function apiRequest(path, options = {}) {
	const { headers, ...rest } = options;
	const response = await fetch(buildApiUrl(path), {
		credentials: "include",
		...rest,
		headers: {
			"Content-Type": "application/json",
			...(headers || {}),
		},
	});

	return parseResponse(response, "Request failed");
}

// Upload a file attachment to a message (multipart/form-data)
export async function uploadFile(messageId, file) {
	const formData = new FormData();
	formData.append("file", file);

	const response = await fetch(buildApiUrl(`/messages/${messageId}/attachments`), {
		method: "POST",
		credentials: "include",
		body: formData,
		// Do NOT set Content-Type — browser sets it with correct boundary
	});

	return parseResponse(response, "Upload failed");
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

// --- Farewell Letters ---

export async function listFarewellLetters(messageId) {
	return apiRequest(`/messages/${messageId}/farewell-letters`);
}

export async function createFarewellLetter(messageId, data) {
	return apiRequest(`/messages/${messageId}/farewell-letters`, {
		method: "POST",
		body: JSON.stringify(data),
	});
}

export async function updateFarewellLetter(messageId, letterId, data) {
	return apiRequest(`/messages/${messageId}/farewell-letters/${letterId}`, {
		method: "PUT",
		body: JSON.stringify(data),
	});
}

export async function deleteFarewellLetter(messageId, letterId) {
	return apiRequest(`/messages/${messageId}/farewell-letters/${letterId}`, {
		method: "DELETE",
	});
}

export async function cancelFarewellLetter(messageId, letterId) {
	return apiRequest(`/messages/${messageId}/farewell-letters/${letterId}/cancel`, {
		method: "POST",
	});
}

export async function cancelAllPendingFarewellLetters(messageId) {
	return apiRequest(`/messages/${messageId}/farewell-letters/cancel-pending`, {
		method: "POST",
	});
}

export async function uploadFarewellAttachment(messageId, letterId, file) {
	const formData = new FormData();
	formData.append("file", file);

	const response = await fetch(
		buildApiUrl(`/messages/${messageId}/farewell-letters/${letterId}/attachments`),
		{
			method: "POST",
			credentials: "include",
			body: formData,
		}
	);

	return parseResponse(response, "Upload failed");
}

export async function listFarewellAttachments(messageId, letterId) {
	return apiRequest(
		`/messages/${messageId}/farewell-letters/${letterId}/attachments`
	);
}

export async function deleteFarewellAttachment(messageId, letterId, attachmentId) {
	return apiRequest(
		`/messages/${messageId}/farewell-letters/${letterId}/attachments/${attachmentId}`,
		{ method: "DELETE" }
	);
}
