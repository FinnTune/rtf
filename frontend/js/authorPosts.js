import { createPostsTable, showEmptyState, renderPostRows, renderPagination, POSTS_PAGE_SIZE } from "./getAllPosts.js";
import { showMessage } from "./notify.js";
import { setActiveNav } from "./navigation.js";
import { clearActiveCategory } from "./categoryFilter.js";

// Renders one author's posts into the content pane — the destination for
// clicking an author's name on a post card or a single-post view. Not a
// sidebar nav destination, so no nav item is highlighted (mirrors how a
// search result or a single post also leave the sidebar nav unhighlighted).
export function showAuthorPosts(author, offset = 0) {
    document.getElementById('msg').textContent = "";
    document.getElementById('search-input').value = "";
    document.getElementById('clear-search-button').style.display = "none";
    document.getElementById('main-content').innerHTML = "";
    clearActiveCategory();
    setActiveNav(null);

    fetch(`getPostsByAuthor?author=${encodeURIComponent(author)}&limit=${POSTS_PAGE_SIZE}&offset=${offset}`, {
        method: 'GET',
        mode: 'cors',
        headers: {
            'Content-Type': 'application/json'
        }
    }).then((response) => {
        if (response.ok) {
            createPostsTable(`Posts by ${author}`);
            const total = parseInt(response.headers.get('X-Total-Count') || '0', 10);
            return response.json().then((posts) => ({ posts, total }));
        }
        return response.text().then((message) => {
            throw new Error(message || `Failed to load posts (${response.status})`);
        });
    }).then(({ posts, total }) => {
        if (offset > 0 && posts.length === 0) {
            // The page we asked for is now empty (e.g. the last post on it
            // was deleted) — snap back to the previous page.
            showAuthorPosts(author, Math.max(0, offset - POSTS_PAGE_SIZE));
            return;
        }
        let postList = document.getElementById('posts-table');
        if (posts.length === 0) {
            showEmptyState(postList, `${author} hasn't posted yet.`);
        } else {
            renderPostRows(postList, posts);
        }
        renderPagination(offset, POSTS_PAGE_SIZE, total, (newOffset) => showAuthorPosts(author, newOffset));
    }).catch((error) => {
        showMessage("Err: " + error.message, "error");
        console.log("Err: ", error);
    });
}
