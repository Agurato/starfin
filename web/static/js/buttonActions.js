// function request(el) {
//   let imdbId = el.getAttribute("tmdbId");
//   let mediaType = el.getAttribute("mediaType");
//   let action = el.getAttribute("action");
//   let url = "/" + mediaType.toLowerCase() + "/" + imdbId + "/" + action;

//   fetch(url, {
//     method: "POST",
//   }).then((res) => {
//     if (res.status == 200) {
//       res.json().then((data) => {
//         console.log(data.message);
//         switch (action) {
//           case "add":
//             el.innerHTML = "Un-request";
//             el.setAttribute("action", "remove");
//             break;
//           case "remove":
//             el.innerHTML = "Request";
//             el.setAttribute("action", "add");
//             break;
//         }
//       });
//     } else {
//       res.json().then((data) => {
//         console.error(res.status, data.message);
//       });
//     }
//   });
// }

function deleteVolume(el) {
  let volumeId = el.getAttribute("volumeId");
  let url = "/admin/deletevolume";

  fetch(url, {
    method: "POST",
    body: new URLSearchParams({ "volumeId": volumeId }),
  }).then((res) => {
    if (res.status == 200) {
      res.json().then((data) => {
        if (data.error) {
          console.error(data.error);
        } else {
          console.log(data.message);
          el.parentNode.parentNode.parentNode.removeChild(
            el.parentNode.parentNode,
          );
        }
      });
    } else {
      res.json().then((data) => {
        console.error(res.status, data.message);
      });
    }
  });
}

function deleteUser(el) {
  let userId = el.getAttribute("userId");
  let url = "/admin/deleteuser";

  fetch(url, {
    method: "POST",
    body: new URLSearchParams({ "userId": userId }),
  }).then((res) => {
    if (res.status == 200) {
      res.json().then((data) => {
        if (data.error) {
          console.error(data.error);
        } else {
          console.log(data.message);
          el.parentNode.parentNode.parentNode.removeChild(
            el.parentNode.parentNode,
          );
        }
      });
    } else {
      res.json().then((data) => {
        console.error(res.status, data.message);
      });
    }
  });
}

function retFalse() {
  return false;
}

function reloadCache(el) {
  let url = "/admin/reloadcache";

  el.setAttribute("disabled", "");
  let spinner = el.children.item(0);
  spinner.style.display = "inline-block";

  fetch(url, {
    method: "POST",
  }).then((res) => {
    if (res.status == 200) {
      res.json().then((data) => {
        if (data.error) {
          console.error(data.error);
        } else {
          console.log(data.message);
          el.removeAttribute("disabled");
          spinner.style.display = "none";
        }
      });
    } else {
      res.json().then((data) => {
        console.error(res.status, data.message);
      });
    }
  });
}

function editFilmOnlineButton(el) {
  let url = "/admin/editfilmonline";

  fetch(url, {
    method: "POST",
    body: new URLSearchParams({
      "url": el.parentNode.parentNode.querySelector("#filmUrl").value,
      "filmID": el.getAttribute("film-id"),
    }),
  }).then((res) => {
    if (res.status == 200) {
      res.json().then((data) => {
        if (data.error) {
          console.error(data.error);
        } else {
          console.log(data);
          location.reload();
          // el.removeAttribute("disabled");
          // spinner.style.display = "none";
        }
      });
    } else {
      res.json().then((data) => {
        console.error(res.status, data.message);
      });
    }
  });
}

function editFilmManualButton(el) {
  for(tag of filmEditGenreTags.getTagElms()) {
    console.log(tag.getAttribute("title"));
  }
  for(tag of filmEditCountryTags.getTagElms()) {
    console.log(tag.getAttribute("code"));
  }
}

function addDirectorLine(el) {
  console.log(el);
}

function deleteDirectorLine(el) {
  console.log(el);
}