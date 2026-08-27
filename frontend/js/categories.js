// Fetches the category list from the server and caches it for the life of
// the page, since both the category filter and the post-creation category
// picker need it and it rarely (if ever) changes during a session.
let categoriesPromise = null;

export function getCategories() {
    if (!categoriesPromise) {
        categoriesPromise = fetch('/getCategories', { method: 'GET' })
            .then((response) => {
                if (!response.ok) {
                    return response.text().then((message) => {
                        throw new Error(message || `Failed to load categories (${response.status})`);
                    });
                }
                return response.json();
            })
            .catch((error) => {
                // Don't cache a failure — let the next caller retry.
                categoriesPromise = null;
                throw error;
            });
    }
    return categoriesPromise;
}
