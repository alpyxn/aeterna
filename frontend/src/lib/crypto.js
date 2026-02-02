export async function generateKey() {
    const key = await window.crypto.subtle.generateKey(
        {
            name: "AES-GCM",
            length: 256,
        },
        true,
        ["encrypt", "decrypt"]
    );
    const exported = await window.crypto.subtle.exportKey("jwk", key);
    // Return readable string (base64url of k) or full JWK?
    // JWK is easiest to handle.
    return exported;
}

export async function importKey(jwk) {
    return await window.crypto.subtle.importKey(
        "jwk",
        jwk,
        { name: "AES-GCM" },
        true,
        ["encrypt", "decrypt"]
    );
}

export async function encryptMessage(text, jwk) {
    const key = await importKey(jwk);
    const iv = window.crypto.getRandomValues(new Uint8Array(12));
    const encoded = new TextEncoder().encode(text);

    const ciphertext = await window.crypto.subtle.encrypt(
        {
            name: "AES-GCM",
            iv: iv,
        },
        key,
        encoded
    );

    // Combine IV + Ciphertext
    const buffer = new Uint8Array(iv.byteLength + ciphertext.byteLength);
    buffer.set(iv, 0);
    buffer.set(new Uint8Array(ciphertext), iv.byteLength);

    // Return base64 string
    return btoa(String.fromCharCode(...buffer));
}

export async function decryptMessage(encryptedBase64, jwk) {
    const key = await importKey(jwk);

    // Decode base64
    const binaryString = atob(encryptedBase64);
    const bytes = new Uint8Array(binaryString.length);
    for (let i = 0; i < binaryString.length; i++) {
        bytes[i] = binaryString.charCodeAt(i);
    }

    const iv = bytes.slice(0, 12);
    const ciphertext = bytes.slice(12);

    const decrypted = await window.crypto.subtle.decrypt(
        {
            name: "AES-GCM",
            iv: iv,
        },
        key,
        ciphertext
    );

    return new TextDecoder().decode(decrypted);
}

// Helper to encode/decode JWK to URL fragment string
export function keyToString(jwk) {
    return btoa(JSON.stringify(jwk));
}

export function stringToKey(str) {
    return JSON.parse(atob(str));
}
