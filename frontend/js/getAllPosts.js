export function getAllPosts() {
    console.log("Getting all posts.")
    fetch('getAllPosts', {
        method: 'GET',
        mode: 'cors',
        headers: {
            'Content-Type': 'application/json'
        }}
    ).then((response) => {
        if(response.ok){
        console.log("Received all posts.")
        // Arrange posts in descending order by date created
        let posts = response.json();
        createPostsTable();
        console.log("PostsBef:", posts);
        return posts;
        }
    }).then((posts) => {
        console.log("PostsAft: ", posts)
        posts.sort((a, b) => (a.CreatedAt > b.CreatedAt) ? -1 : 1);
        let table = document.getElementById('posts-table');
        let tbody = table.querySelector('tbody');
        for(let i = 0; i < posts.length; i++){
            let row = tbody.insertRow();
            let title = row.insertCell(0);
            let content = row.insertCell(1);
            let author = row.insertCell(2);
            let dateCreated = row.insertCell(3);
            let link = document.createElement("a");
            link.href = "/posts/" + posts[i].Id;
            link.className = "post-link";
            link.innerHTML = posts[i].Title;
            link.addEventListener("click", function(event){
                event.preventDefault();
                displaySinglePost(posts[i]);
            });
            title.appendChild(link);
            content.innerHTML = posts[i].Content;
            author.innerHTML = posts[i].UserId;
            dateCreated.innerHTML = posts[i].Created;
    }}).catch((error) => {
        alert("Err: " + error);
        console.log("Err: ", error);
    });
 
    return false;
}

function createPostsTable() {
    // Get the main content element
    const mainContent = document.getElementById('main-content');
  
    // Create the posts div element
    const postsDiv = document.createElement('div');
    postsDiv.setAttribute('id', 'posts');
  
    // Create the table element
    const table = document.createElement('table');
    table.setAttribute('id', 'posts-table');
  
    // Create the table header row and cells
    const headerRow = document.createElement('tr');
    const titleHeader = document.createElement('th');
    titleHeader.textContent = 'Title';
    const contentHeader = document.createElement('th');
    contentHeader.textContent = 'Content';
    const authorHeader = document.createElement('th');
    authorHeader.textContent = 'Author';
    const createdHeader = document.createElement('th');
    createdHeader.textContent = 'Created';
  
    // Append the cells to the header row
    headerRow.appendChild(titleHeader);
    headerRow.appendChild(contentHeader);
    headerRow.appendChild(authorHeader);
    headerRow.appendChild(createdHeader);
  
    // Create the table body element
    const tbody = document.createElement('tbody');
  
    // Append the header row to the table body
    tbody.appendChild(headerRow);
  
    // Append the table body to the table
    table.appendChild(tbody);
  
    // Append the table to the posts div
    postsDiv.appendChild(table);
  
    // Append the posts div to the main content element
    mainContent.appendChild(postsDiv);
  }
  
  

function displaySinglePost(post) {
    let mainContent = document.getElementById("main-content");
    let singlePostDiv = document.createElement("div");
    singlePostDiv.id = "single-post";
    let title = document.createElement("h3");
    title.innerHTML = post.Title;
    let content = document.createElement("p");
    content.innerHTML = post.Content;
    let author = document.createElement("p");
    author.innerHTML = "Author: " + post.UserId;
    let dateCreated = document.createElement("p");
    dateCreated.innerHTML = "Created: " + post.Created;
    let backButton = document.createElement("button");
    backButton.className = "btns";
    backButton.innerHTML = "Back to Posts";
    backButton.addEventListener("click", function(event){
        event.preventDefault();
        mainContent.innerHTML = "";
        mainContent.appendChild(document.createElement("h3")).innerHTML = "Latest Posts";
        getAllPosts();
    });
    singlePostDiv.appendChild(title);
    singlePostDiv.appendChild(content);
    singlePostDiv.appendChild(author);
    singlePostDiv.appendChild(dateCreated);
    singlePostDiv.appendChild(backButton);
    mainContent.innerHTML = "";
    mainContent.appendChild(singlePostDiv);
}