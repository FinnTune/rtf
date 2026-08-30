import { createPostsTable, showEmptyState, renderPostRows } from "./getAllPosts.js";
import { showMessage, setButtonLoading } from "./notify.js";
import { setActiveNav } from "./navigation.js";

export function searchPosts(query) {
    const trimmed = query.trim();
    if (!trimmed) {
        return;
    }

    const submitButton = document.getElementById('search-submit-button');
    setButtonLoading(submitButton, true, 'Searching...');

    document.getElementById('main-content').innerHTML = "";
    setActiveNav(null);

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
        let postList = document.getElementById('posts-table');
        if (posts.length === 0) {
            showEmptyState(postList, `No posts found for "${trimmed}".`);
        } else {
            renderPostRows(postList, posts);
        }
        document.getElementById('clear-search-button').style.display = 'inline';
    }).catch((error) => {
        showMessage("Err: " + error.message, "error");
        console.error(error);
    }).finally(() => {
        setButtonLoading(submitButton, false);
    });
}
