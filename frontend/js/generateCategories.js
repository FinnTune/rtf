import { getCategories } from "./categories.js";

export async function generateCategoryDropdown() {
  const form = document.getElementById('categories');
  if (form == null) {
    return;
  }

  let categories;
  try {
    categories = await getCategories();
  } catch (error) {
    console.error(error);
    return;
  }

  var dropdown = document.createElement('div');
  dropdown.className = 'dropdown';

  var dropdownToggle = document.createElement('span');
  dropdownToggle.textContent = 'Select Categories>>';
  dropdownToggle.addEventListener('click', function() {
    dropdownContent.style.display = (dropdownContent.style.display === 'block') ? 'none' : 'block';
  });
  dropdown.appendChild(dropdownToggle);

  var dropdownContent = document.createElement('div');
  dropdownContent.className = 'dropdown-content';

  categories.forEach(function(category) {
    var checkbox = document.createElement('input');
    checkbox.type = 'checkbox';
    checkbox.name = 'categories[]';
    checkbox.title = category.id;
    checkbox.value = category.name;

    var label = document.createElement('label');
    label.appendChild(checkbox);
    label.appendChild(document.createTextNode(category.name));
    dropdownContent.appendChild(label);
  });

  dropdown.appendChild(dropdownContent);
  form.appendChild(dropdown);
}
