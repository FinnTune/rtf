export function generateCategoryDropdown() {
    var categories = [
        { id: 1, name: 'Cuisine' },
        { id: 2, name: 'Places' },
        { id: 3, name: 'Activities' },
        { id: 4, name: 'Events' },
        { id: 5, name: 'Code' },
        { id: 6, name: 'Language' },
        { id: 7, name: 'Sports' },
        { id: 8, name: 'Politics' },
        { id: 9, name: 'Social' },
        { id: 10, name: 'Religion' },
        { id: 11, name: 'Business' },
        { id: 12, name: 'Geography' },
        { id: 13, name: 'Science' },
        { id: 14, name: 'Health' },
        { id: 15, name: 'Other' }
      ];

      var form = document.getElementById('categories');
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