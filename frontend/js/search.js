import { createPostsTable, showEmptyState, renderPostRows } from "./getAllPosts.js";
import { showMessage, setButtonLoading } from "./notify.js";

export function searchPosts(query) {
    const trimmed = query.trim();
    if (!trimmed) {
        return;
    }

    const submitButton = document.getElementById('search-submit-button');
    setButtonLoading(submitButton, true, 'Searching...');

    document.getElementById('main-content').innerHTML = "";

    fetch(`/searchPosts?q=${encodeURIComponent(trimmed)}`, {
        method: 'GET',
        mode: 'cors',
        headers: {
            'Content-Type': 'application/json'
        }
    }).then((response) => {
        if (response.ok) {
            createPostsTable();
            return response.json();
        }
        return response.text().then((message) => {
            throw new Error(message || `Search failed (${response.status})`);
        });
    }).then((posts) => {
        let table = document.getElementById('posts-table');
        let tbody = table.querySelector('tbody');
        if (posts.length === 0) {
            showEmptyState(tbody, `No posts found for "${trimmed}".`);
        } else {
            renderPostRows(tbody, posts);
        }
        document.getElementById('clear-search-button').style.display = 'inline';
    }).catch((error) => {
        showMessage("Err: " + error.message, "error");
        console.log("Err: ", error);
    }).finally(() => {
        setButtonLoading(submitButton, false);
    });
}
