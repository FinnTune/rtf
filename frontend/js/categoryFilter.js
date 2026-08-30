import { createPostsTable, getAllPosts, showEmptyState, renderPostRows, renderPagination, POSTS_PAGE_SIZE } from "./getAllPosts.js";
import { clearTable } from "./getAllPosts.js";
import { showMessage } from "./notify.js";
import { getCategories } from "./categories.js";

// The single category currently selected in the sidebar nav, or null when
// browsing all posts. Clicking a category is a nav destination (like "All
// Posts"), not a multi-select filter, so there's exactly one active value.
let activeCategory = null;

export function clearActiveCategory() {
    activeCategory = null;
}

export async function createCategoryFilter() {
    const categoryNavDiv = document.getElementById('category-selection');
    categoryNavDiv.replaceChildren();

    let categories;
    try {
        categories = await getCategories();
    } catch (error) {
        showMessage("Err: " + error.message, "error");
        console.error(error);
        return categoryNavDiv;
    }

    for (let category of categories) {
        const item = document.createElement('button');
        item.type = 'button';
        item.className = 'nav-item category-nav-item';
        item.textContent = category.name;
        item.dataset.categoryName = category.name;
        if (category.name === activeCategory) {
            item.classList.add('active');
        }

        item.addEventListener('click', () => {
            activeCategory = category.name;
            document.querySelectorAll('#all-posts-button, #create-post-button, .category-nav-item')
                .forEach((el) => el.classList.remove('active'));
            item.classList.add('active');
            getPostsByCategory();
        });

        categoryNavDiv.appendChild(item);
    }

    return categoryNavDiv;
}

export function getPostsByCategory(offset = 0) {
    document.getElementById('main-content').innerHTML = "";
    if (!activeCategory) {
        return getAllPosts();
    }

     fetch(`getPostsByCategory?limit=${POSTS_PAGE_SIZE}&offset=${offset}`, {
         method: 'POST',
         mode: 'cors',
         headers: {
             'Content-Type': 'application/json'
         },
         body: JSON.stringify({ categories: [activeCategory] }) // Send the selected category to the server
     }).then((response) => {
        if(response.ok){
            if (document.getElementById('posts')) {
                clearTable();
            } else {
                createPostsTable();
            }
            const total = parseInt(response.headers.get('X-Total-Count') || '0', 10);
            return response.json().then((posts) => ({ posts, total }));
        }
        return response.text().then((message) => {
            throw new Error(message || `Failed to load posts (${response.status})`);
        });
    }).then(({ posts, total }) => {
        // Posts already arrive newest-first from the server.
        if (offset > 0 && posts.length === 0) {
            // The page we asked for is now empty (e.g. the last post on it
            // was deleted) — snap back to the previous page.
            getPostsByCategory(Math.max(0, offset - POSTS_PAGE_SIZE));
            return;
        }
        let postList = document.getElementById('posts-table');
        if (posts.length == 0) {
            showEmptyState(postList, "No posts for this category.");
        } else {
            renderPostRows(postList, posts);
        }
        renderPagination(offset, POSTS_PAGE_SIZE, total, getPostsByCategory);
    }).catch((error) => {
        showMessage("Err: " + error.message, "error");
        console.error(error);
    });
    return
}
