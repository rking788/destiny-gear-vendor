window.addEventListener("load", enableSearch);
document.getElementById("search-text").addEventListener("input", textChanged);

function enableSearch() {
    const searchText = document.getElementById('search-text');
    searchText.removeAttribute("disabled");
}

function textChanged() {
    const search = document.getElementById('search-text').value.toLowerCase();

    let gear = document.getElementsByClassName('item');

    for (let i = 0; i < gear.length; i++) {
        const gearPiece = gear[i];
        const name = gearPiece.getAttribute('data-name');

        if (name.includes(search)) {
            gearPiece.style.display = "";
        } else {
            gearPiece.style.display = "none";
        }
    }
}