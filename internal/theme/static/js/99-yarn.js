var textMaxLength = ""; // Previous value of maxlength of #text
var textRows = ""; // Previous value of rows of #text

var $mentionedList = u("#mentioned-list").first(); // node list of mentioned users
var lastSymbol = ""; // last char in textarea

// Array.findIndex polyfill
if (!Array.prototype.findIndex) {
  Array.prototype.findIndex = function (predicate) {
    if (this == null) {
      throw new TypeError(
        "Array.prototype.findIndex called on null or undefined"
      );
    }
    if (typeof predicate !== "function") {
      throw new TypeError("predicate must be a function");
    }
    var list = Object(this);
    var length = list.length >>> 0;
    var thisArg = arguments[1];
    var value;

    for (var i = 0; i < length; i++) {
      value = list[i];
      if (predicate.call(thisArg, value, i, list)) {
        return i;
      }
    }
    return -1;
  };
}

if (!Array.prototype.find) {
  Object.defineProperty(Array.prototype, "find", {
    value: function (predicate) {
      // 1. Let O be ? ToObject(this value).
      if (this == null) {
        throw new TypeError('"this" is null or not defined');
      }

      var o = Object(this);

      // 2. Let len be ? ToLength(? Get(O, "length")).
      var len = o.length >>> 0;

      // 3. If IsCallable(predicate) is false, throw a TypeError exception.
      if (typeof predicate !== "function") {
        throw new TypeError("predicate must be a function");
      }

      // 4. If thisArg was supplied, let T be thisArg; else let T be undefined.
      var thisArg = arguments[1];

      // 5. Let k be 0.
      var k = 0;

      // 6. Repeat, while k < len
      while (k < len) {
        // a. Let Pk be ! ToString(k).
        // b. Let kValue be ? Get(O, Pk).
        // c. Let testResult be ToBoolean(? Call(predicate, T, « kValue, k, O »)).
        // d. If testResult is true, return kValue.
        var kValue = o[k];
        if (predicate.call(thisArg, kValue, k, o)) {
          return kValue;
        }
        // e. Increase k by 1.
        k++;
      }

      // 7. Return undefined.
      return undefined;
    },
    configurable: true,
    writable: true,
  });
}

function reply(e) {
  e.preventDefault();

  movePostBox(e);

  var el = u("textarea#text");
  var text = u("#text").first();

  el.empty();

  text.value = u(e.target).data("text");
  text.focus();

  var size = text.value.length;
  text.setSelectionRange(size, size);

  u("#subject").first().value = u(e.target).data("subject");
}
fork = reply;

function editTwt(e) {
  e.preventDefault();

  movePostBox(e);

  var el = u("textarea#text");
  var text = u("#text").first();

  el.empty();

  text.value = u(e.target).data("text");
  text.focus();

  var size = text.value.length;
  text.setSelectionRange(size, size);

  u("#hash").first().value = u(e.target).data("hash");
  u("#subject").first().value = u(e.target).data("subject");
}

function movePostBox(e) {
  e.preventDefault();
  var hr = window.location.href;

  if (hr.indexOf('/user/') > 0 || hr.indexOf('/external?') > 0) {
    if (hr.indexOf('/bookmarks') > 0) {
      u("#newPost").first().style.display = "none";
    } else {
      u("#postToUser").first().style.display = "none";
    }
  } else if (hr.indexOf('/conv/') == -1) {
    if (!(u("#postbox").hasClass("single-twt"))) {
      u("#newPost").first().style.display = "none";
    }
  }

  u("article").each(function (n, i) {
    u(n).removeClass("highlight");
  });

  var yarn = u(e.target).closest(".twt-nav");
  var article = yarn.parent();
  var postbox = u("#postbox").clone();

  article.addClass("highlight");

  u("#postbox").remove();
  yarn.after(postbox);
  htmx.process(postbox.first());
  postbox.addClass("drawer");

  u("#toolbar").addClass("toolbar-reply");
  u("#form").addClass("form-reply");

}

function clearPostBox() {
  localStorage.removeItem("text");
  u("textarea#text").first().value = "";
}

function resetPostBox() {
  if (!u("#form").hasClass("form-reply")) {
    return;
  }

  u('article').each(function (n) {
    u(n).removeClass('highlight');
  });

  var postbox = u("#postbox").clone();
  
  var pbclass = u("#postbox").hasClass("single-twt, drawer");
  postbox.removeClass("drawer");

  u("#postbox").remove();
  var hr = window.location.href;

  if (hr.indexOf('/user/') > 0 || hr.indexOf('/external?') > 0) {
    if (hr.indexOf('/bookmarks') > 0) {
      u("#newPost").first().style.display = "initial";
      u("#newPost").prepend(postbox);
    } else {
      u("#postToUser").first().style.display = "initial";
      u("#postToUser").prepend(postbox);
    }
  } else if (hr.indexOf('/conv/') > 0 || pbclass) {
    u("main").append(postbox);
  } else {
    u("#newPost").first().style.display = "initial";
    u("#newPost").prepend(postbox);
  }
  htmx.process(postbox.first())

  u("#toolbar").removeClass("toolbar-reply");
  u("#form").removeClass("form-reply");

  u("#postbox").scroll();
}

u.prototype.isHidden = function () {
  var e = this.first();
  return (e.offsetParent === null)
};

u.prototype.getSelection = function () {
  var e = this.first();

  return (
    /* mozilla / dom 3.0 */
    (
      ("selectionStart" in e &&
        function () {
          var l = e.selectionEnd - e.selectionStart;
          return {
            start: e.selectionStart,
            end: e.selectionEnd,
            length: l,
            text: e.value.substr(e.selectionStart, l),
          };
        }) ||
      /* exploder */
      (document.selection &&
        function () {
          e.focus();

          var r = document.selection.createRange();
          if (r === null) {
            return {
              start: 0,
              end: e.value.length,
              length: 0
            };
          }

          var re = e.createTextRange();
          var rc = re.duplicate();
          re.moveToBookmark(r.getBookmark());
          rc.setEndPoint("EndToStart", re);

          return {
            start: rc.text.length,
            end: rc.text.length + r.text.length,
            length: r.text.length,
            text: r.text,
          };
        }) ||
      /* browser not supported */
      function () {
        return null;
      }
    )()
  );
};

u.prototype.replaceSelection = function () {
  var e = this.first();

  var text = arguments[0] || "";

  return (
    /* mozilla / dom 3.0 */
    (
      ("selectionStart" in e &&
        function () {
          e.value =
            e.value.substr(0, e.selectionStart) +
            text +
            e.value.substr(e.selectionEnd, e.value.length);
          return this;
        }) ||
      /* exploder */
      (document.selection &&
        function () {
          e.focus();
          document.selection.createRange().text = text;
          return this;
        }) ||
      /* browser not supported */
      function () {
        e.value += text;
        return jQuery(e);
      }
    )()
  );
};

function createMentionedUserNode(match) {
  return u("<div>")
    .addClass("user-list__user")
    .append(
      u("<div>")
        .addClass("avatar")
        .attr(
          "style",
          "background-image: url('" + match.Avatar + "')"
        )
    )
    .append(
      u("<div>")
        .addClass("info")
        .append(u("<div>").addClass("nickname").text(match.Nick.replace(/@.+/g, "")))
        .append(u("<div>").addClass("domain").text(match.Domain))
    );
}

function formatText(selector, fmt, single = false) {
  selector.first().focus();

  var finalText = "";
  var start = selector.first().selectionStart;
  var selectedText = selector.getSelection().text;

  if (fmt.includes("$title")) {
    if (selectedText.length == 0) {
      finalText = fmt;
    } else {
      finalText = fmt.replace("$title", selectedText);
    }
  } else if (single) {
    finalText = fmt;
  } else {
    if (selectedText.length == 0) {
      finalText = fmt + fmt;
    } else {
      finalText = fmt + selectedText + fmt;
    }
  }

  selector.replaceSelection(finalText, true);
  selector.first().focus();
  if (!selectedText.length) {
    var selectionRange = start + fmt.length;
    selector.first().setSelectionRange(selectionRange, selectionRange);
  }
}

function insertText(selector, text) {
  var start = selector.first().selectionStart;

  selector.first().value.slice(startMention, start);
  selector.replaceSelection(text, false);
  selector.first().focus();

  var selectionRange =
    selector.first().value.substr(start + text.length - 1, 1) === ")" ?
      start + text.length - 1 :
      start + text.length;

  selector.first().setSelectionRange(selectionRange, selectionRange);
}

function iOS() {
  return (
    [
      "iPad Simulator",
      "iPhone Simulator",
      "iPod Simulator",
      "iPad",
      "iPhone",
      "iPod",
    ].indexOf(navigator.platform) !== -1 ||
    // iPad on iOS 13 detection
    (navigator.userAgent.indexOf("Mac") !== -1 && "ontouchend" in document)
  );
}

function IE() {
  return !!window.MSInputMethodContext && !!document.documentMode;
}

var deBounce = 300;
var fetchUsersTimeout = null;
var startMention = null;

function getUsers(searchStr) {
  clearTimeout(fetchUsersTimeout);
  fetchUsersTimeout = setTimeout(function () {
    let requestUrl = "/lookup";

    if (searchStr) {
      requestUrl += "?prefix=" + searchStr;
    }

    Twix.ajax({
      type: "GET",
      url: requestUrl,
      success: function (data) {
        u("#mentioned-list-content").empty();
        data.map(function (match) {
          u("#mentioned-list-content").append(createMentionedUserNode(match));
        });
        if (data.length) {
          u(".user-list__user").first().classList.add("selected");
        }
      },
    });
  }, deBounce);
}


var maxTaskWait = (1000 * 60 * 10); // ~10mins TODO: Make this configurable

function pollForTask(taskURL, delay, maxDelay, timeout, errorCallback, successCallback) {
  Twix.ajax({
    type: "GET",
    url: taskURL,
    error: function (statusCode, statusText) {
      errorCallback({
        error: statusCode + " " + statusText
      })
    },
    success: function (data) {
      switch (data.state) {
        case "pending":
        case "running":
          if (Date.now() < timeout) {
            if (delay < maxDelay) {
              delay = delay * 2;
            }
            setTimeout(function () {
              pollForTask(taskURL, delay, maxDelay, timeout, errorCallback, successCallback);
            }, delay);
            return;
          }
          break;
        case "complete":
          successCallback(data);
          break;
        default:
          errorCallback(data);
      }
    },
  });
}

function clearMentionedList() {
  u("#mentioned-list").attr("style", "display: none;");
  u("#mentioned-list-content").first().innerHTML = "";
}

function showMentionedList() {
  u("#mentioned-list").attr("style", "display: inherit;");
  u("#mentioned-list").first().style.top =
    u("textarea#text").first().clientHeight + 2 + "px";
}

function toggleReadMore(e) {
  var button = u(e.target);
  var content = button.parent().find(".e-content");
  if (button.text() == "⤋ Read More") {
    content.removeClass("p-compact");
    button.text("⤊ Read Less");
  } else {
    content.addClass("p-compact");
    button.text("⤋ Read More");
  }
}

function maybeReadMore(article) {
  if (!hasPref("readmore")) {
    return;
  }
  var content = u(article).find(".e-content");
  var button = u(article).find(".read-more");
  if (Math.ceil(content.size().height) > 225) {
    button.first().style.display = "inline-block";
    content.addClass("p-compact");
  }
}

if (window.performance.getEntriesByType("navigation")[0].type === "back_forward") {
  window.scrollTo(0, Number(localStorage.getItem("prevOffset")));
}

window.addEventListener("scroll", function () {
  if (u("#mobileMenuInput").is(":checked")) {
    u("#mobileMenuInput").first().checked = false;
  }
});

window.addEventListener("blur", function () {
  u("article").find("dialog").each(function (modal, i) {
    if (u(modal).attr("open") == "true") {
      if (u(modal).attr("id").startsWith("lvm-")) {
        u(modal).trigger("click");
      } else {
        u(modal).find("img").trigger("click");
      }
    }
  });
});

window.onbeforeunload = function () {
  if (u("textarea#text").length > 0) {
    var posttext = u("textarea#text").first().value;
    if (posttext.length > 0 && localStorage.getItem("isPost") == "false") {
      localStorage.setItem("text", posttext);
    }
  }
  localStorage.setItem("prevOffset",
    localStorage.getItem("currentOffset") || String(window.scrollY)
  );
  localStorage.setItem("currentOffset", String(window.scrollY));
};

function stripBookmarkletParams(uri) {
  var url = new URL(uri);
  var params = new URLSearchParams(url.search);

  params.delete('title');
  params.delete('url');

  url.search = "?" + params.toString();

  return url.toString();
}

function stripTrackingParameters(uri) {
  if (uri.startsWith("/")) {
    return uri;
  }

  var url = new URL(uri.toString());
  var params = new URLSearchParams(url.search);

  if (params.toString().length > 0) {
    const re = /(((yc|fc|dc|fbc|gcl)(lid|src)|gaa|amp|mkt_tok|utm|ar|si|sc|fb)([_\-a-z0-9=\.]+))(\&?)/g
    const href = url.origin + url.pathname + "?";
    const query = params.toString().replace(re, "").replace(/&$/g, "");
    uri = (href + query).toString();
  }

  return uri;
}

function hasPref(pref) {
  data = u("prefs").attr("data-" + pref);

  if (pref == "openlink" && data == "newwindow") {
    return "_blank";
  } else if (pref == "openlink") {
    return "_self";
  }

  // This is dumb, but for some reason won't work otherwise?!
  if (data === "true") {
    return true;
  } else {
    return false;
  }
}

function usedBookmarklet() {
  if (typeof (window.URLSearchParams) != "undefined") {
    var urlParams = new URLSearchParams(window.location.search);
    var titleParam = urlParams.get("title");
    var urlParam = urlParams.get("url");

    if (titleParam && urlParam) {
      var bookmarklet = " [" + titleParam + "](" + urlParam + ")"
      var text = localStorage.getItem('text') || "";
      localStorage.setItem("text", text + bookmarklet);
    }
  }
}

function closeMobileToggle() {
  if (u("#mobileToggle").first().checked) {
    u("#mobileToggle").first().checked = false
  }
}

function waitForElm(selector) {
  return new Promise(resolve => {
    if (document.querySelector(selector)) {
      return resolve(document.querySelector(selector));
    }

    const observer = new MutationObserver(function () {
      if (document.querySelector(selector)) {
        resolve(document.querySelector(selector));
        observer.disconnect();
      }
    });

    observer.observe(document.body, {
      childList: true,
      subtree: true
    });
  });
}

function enableFormattingToolbar(el) {
  u(el).find("#bBtn").on("click", function (e) {
    e.preventDefault();
    formatText(u("textarea#text"), "**");
  });

  u(el).find("#iBtn").on("click", function (e) {
    e.preventDefault();
    formatText(u("textarea#text"), "_");
  });

  u(el).find("#sBtn").on("click", function (e) {
    e.preventDefault();
    formatText(u("textarea#text"), "~~");
  });

  u(el).find("#cBtn").on("click", function (e) {
    e.preventDefault();
    formatText(u("textarea#text"), "`");
  });

  u(el).find("#bqBtn").on("click", function (e) {
    e.preventDefault();
    formatText(u("textarea#text"), "\n>");
  });

  u(el).find("#lnkBtn").on("click", function (e) {
    e.preventDefault();
    formatText(u("textarea#text"), "[$title](https://)");
  });

  u(el).find("#imgBtn").on("click", function (e) {
    e.preventDefault();
    formatText(u("textarea#text"), '![](https:// "$title")');
  });

  u(el).find("#usrBtn").on("click", function (e) {
    e.preventDefault();
    u("textarea#text").first().focus();
    startMention = u("textarea#text").first().selectionStart + 1;
    insertText(u("textarea#text"), "@");
    showMentionedList();
    getUsers();
  });
}

function setupPostBox(el) {
  enableFormattingToolbar(el);

  /* Support drag 'n drop of media to upload */
  u(el).on("drop", function (e) {
    e.preventDefault();
    var uploadMedia = document.getElementById("uploadMedia");
    uploadMedia.files = e.dataTransfer.files;
    u("#uploadMedia").trigger("change");
  });

  u(el).find("textarea#text").on("focus", function (e) {
    /* Show the @-mention user list */
    if (e.relatedTarget === u("#usrBtn").first()) {
      showMentionedList();
      getUsers();
    }
  }).on("keydown", function (e) {
    /* Keyboard Shortcuts */
    if (e.metaKey && e.key == "b") {
      u("#bBtn").trigger("click");
    } else if (e.metaKey && e.key == "i") {
      u("#iBtn").trigger("click");
    }
  });

  /* Preserve the text to localStorage */
  u(el).find("textarea#text").on('change click blur paste', function () {
    localStorage.setItem("text", u("textarea#text").first().value.trim());
  });

  /*Auto re-size the textarea whilst preserving the original height.

  NOTE: There is the CSS resize attribute but this is not supported on Mobile Safari :/
  */
  var _ = function() {
    var text = u("textarea#text").first();
    var size = u("textarea#text").size();
    u(el).find("textarea#text").on("input change paste", function () {
      text.style.height = (size.height) + "px";
      text.style.height = (text.scrollHeight) + "px";
    });
  }();

  /* Show/Hide the @-mention user list */
  u(el).find("textarea#text").on("keyup", function (e) {
    if (e.key.length === 1 || e.key === "Backspace") {
      var idx = e.target.selectionStart;
      var prevSymbol = e.target.value.slice(idx - 1, idx);

      if (prevSymbol === "@") {
        startMention = idx;
        showMentionedList();
      }

      if (!u("#mentioned-list").isHidden()) {
        var searchStr = e.target.value.slice(startMention, idx);
        if (!prevSymbol.trim()) {
          clearMentionedList();
          startMention = null;
        } else {
          getUsers(searchStr);
        }
      }
    }
  });

  /* Change the selected @-mention on mousemove */
  u(el).find("#mentioned-list-content").on("mousemove", function (e) {
    var target = e.target;
    u(".user-list__user").nodes.forEach(function (item) {
      item.classList.remove("selected");
    });
    if (target.classList.contains("user-list__user")) {
      target.classList.add("selected");
    }
  });

  /* Complete the @-mention on click */
  u(el).find("#mentioned-list").on("click", function (e) {
    var value = u("textarea#text").first().value;

    u("textarea#text").first().value =
      value.slice(0, startMention) +
      value.slice(u("textarea#text").first().selectionEnd);

    u("textarea#text").first().setSelectionRange(startMention, startMention);
    insertText(u("textarea#text"), e.target.innerText.replace('\n', '@').trim());
    u("#mentioned-list").attr("style", "display: none;");
  });

  u(el).find("#uploadMedia").on("change", function (e) {
    u("#uploadMediaButton").attr("aria-busy", true);
    u("#uploadMediaForm").data("tooltip", "Uploading...");

    document.body.style.cursor = "progress";

    Twix.ajax({
      type: "POST",
      url: u("#mediaUploadForm").attr("action"),
      data: new FormData(u("#mediaUploadForm").first()),
      success: function (data) {
        var el = u("textarea#text");
        var text = document.getElementById("text");

        pollForTask(
          data.Path,
          1000,
          30000,
          Date.now() + maxTaskWait,

          function (errorData) {
            u("#uploadMediaButton").attr("aria-busy", false);
            alert("An error occurred uploading your media: " + errorData.error)
          },

          function (successData) {
            formatText(u("textarea#text"), " ![](" + successData.data.mediaURI + ") ", true);

            el.scroll();
            text.focus();

            var size = el.text().length;
            text.setSelectionRange(size, size);

            u("#uploadMediaButton").attr("aria-busy", false);
            u("#uploadMedia").data("tooltip", "Upload");
          }

        );
      },

      error: function (statusCode, statusText) {
        u("#uploadMediaButton").attr("aria-busy", false);
        alert("An error occurred uploading your media: " + statusCode + " " + statusText);
      },

    });

    document.body.style.cursor = "default";
  });
}

function setupNewPost(el) {
  /* Focus PostBox's TextArea on Toggling NewPost */
  u(el).on("toggle", function (e) {
    if (u(e.target).attr("open") != null) {
      var text = localStorage.getItem('text');
      if (text) {
        u("textarea#text").first().value = text;
      }
      u("textarea#text").first().focus();
    }
  });
}

function onLoaded() {
  /* Close the PostBox on Escape if we moved it */
  u("body").on("keyup", function (e) {
    if (e.key != "Escape") {
      return;
    }

    e.preventDefault();

    // Reset and close the postBox on Esc if replying
    if (u("#text").first().value.length > 0) {
      if (u("#postbox").hasClass("drawer")) {
        if (confirm('Are you sure you want to cancel this reply?')) {
          resetPostBox();
          u("#text").first().value = "";
        }
      } else {
        if (confirm('Are you sure you want to cancel this post?')) {
          u("#text").first().value = "";
        }
      }
    }

    localStorage.setItem("text", "");
    localStorage.setItem("isPost", "false");

    // Reset and close the postBox on Esc if replying
    if (u("#newPost").attr("open") != null) {
      if (u("textarea#text").length > 0) {
        if (!(u("#postbox").is(".single-twt, .yarn-post"))) {
          if (u("#newPost").length > 0) {
            u("#newPost").first().removeAttribute("open");
          } else {
            u("#postToUser").first().removeAttribute("open");
          }
        }
      }
    }
  });

  u("dialog img").on("click", function (e) {
    e.preventDefault();
    if (visibleModal != null) {
      closeModal(visibleModal);
    }
  });

  u("body").on("keydown", function (e) {
    if (!u("#mentioned-list").first() || u("#mentioned-list").isHidden()) {
      return;
    }

    if (e.key === "Escape") {
      clearMentionedList();
    }

    if (
      e.key === "ArrowUp" ||
      e.key === "ArrowDown" ||
      e.key === "Up" ||
      e.key === "Down"
    ) {
      e.preventDefault();

      var selectedIdx = u(".user-list__user").nodes.findIndex(function (
        item
      ) {
        return item.classList.contains("selected");
      });

      var nextIdx;
      var scrollOffset;

      if (e.key === "ArrowDown" || e.key === "Down") {
        nextIdx =
          selectedIdx + 1 === u(".user-list__user").length ?
            0 :
            selectedIdx + 1;
      } else if (e.key === "ArrowUp" || e.key === "Up") {
        nextIdx =
          selectedIdx - 1 < 0 ?
            u(".user-list__user").length - 1 :
            selectedIdx - 1;
      }

      scrollOffset =
        u(".user-list__user").first().clientHeight * (nextIdx - 2);

      u(".user-list__user").nodes.forEach(function (item, index) {
        item.classList.remove("selected");
        if (index === nextIdx) {
          u("#mentioned-list-content").first().scrollTop =
            scrollOffset > 0 ? scrollOffset : 0;
          item.classList.add("selected");
        }
      });
    }

    if (e.key === "Tab" || e.key === "Enter") {
      e.preventDefault();

      var selectedNodeIdx = u(".user-list__user").nodes.findIndex(function (
        item
      ) {
        return item.classList.contains("selected");
      });

      var selectedNode = u(".user-list__user").nodes[selectedNodeIdx];
      var value = u("textarea#text").first().value;

      u("textarea#text").first().value =
        value.slice(0, startMention) +
        value.slice(u("textarea#text").first().selectionEnd);

      u("textarea#text")
        .first()
        .setSelectionRange(startMention, startMention);
      insertText(u("textarea#text"), selectedNode.innerText.replace('\n', '@').trim());
      clearMentionedList();
    }

    var caret = u("textarea#text").first().selectionStart;
    var prevSymbol = u("textarea#text")
      .first()
      .value.slice(caret - 1, 1);

    if (e.key === "Backspace" && prevSymbol === "@") {
      clearMentionedList();
    }
  });

  tippy.setDefaultProps({
    delay: 50,
    trigger: 'click',
    appendTo: document.body,
    allowHTML: true,
  });

  if (window.location.hash != "") {
    waitForElm(window.location.hash).then(function (e) {
      u(e).scroll();
    })
  }
}
document.addEventListener("DOMContentLoaded", onLoaded);

function createTippyInstances() {
  tippy('a#e-media', {
    content(media) {
      var mData = media.getAttribute('href');
      var mTitle = media.getAttribute('data-title');
      if (mData.endsWith('.mp4') || mData.endsWith('.mp4?full=1')) {
        return '<video controls playsinline><source type="video/mp4" src="' + mData + '">Your browser does not support the video element.</video>' + mTitle;
      }
      if (mData.endsWith('.mp3') || mData.endsWith('.mp3?full=1')) {
        return '<audio controls><source type="audio/mp3" src="' + mData + '">Your browser does not support the audio element.</audio>' + mTitle;
      }
      return '<img class="tippy-img" src="' + mData + '">' + mTitle;
    },
    interactive: true,
  });

  tippy('a#v-info', {
    content(info) {
      return info.getAttribute('data-commit');
    },
    trigger: 'mouseenter focus',
    touch: ['hold', 500],
    interactive: true,
  });

  tippy('span.help', {
    content(help) {
      const title = help.getAttribute('title');
      help.removeAttribute('title');
      return title;
    },
    interactive: true,
  });

  tippy('span.vp-d-help', {
    content(help) {
      return '<svg version="1.2" viewBox="0 0 195 125" width="195px" height="125px"><style>tspan{white-space:pre}.t7,.t8{font-size:10px;fill:var(--muted-color);font-family:var(--font-family)}.t8{font-size:11px;fill:var(--primary)}</style><path fill="var(--card-background-color)" d="M.3 0h225v125H.3z"/><path fill="var(--muted-color)" d="M8.3 14.8C8.3 11 11.4 8 15.1 8h18.4c3.8 0 6.8 3 6.8 6.8v18.4c0 3.8-3 6.8-6.8 6.8H15.1c-3.7 0-6.8-3-6.8-6.8z"/><path fill="var(--primary)" d="M43.7 12h90v6h-90zm0 9h90v6h-90z"/><path fill="var(--muted-color)" d="M43.7 30h90v6h-90zM5.3 95c114.1.7 215 0 215 0"/><text transform="translate(8.89 113.02)"><tspan x="0" y="0" class="t8">Reply</tspan></text><text transform="translate(60.77 113.02)"><tspan x="0" y="0" class="t8">Edit</tspan></text><text transform="translate(101.62 113.02)"><tspan x="0" y="0" class="t8">Yarn</tspan></text><text transform="translate(5.34 58.16)"><tspan x="0" y="0" class="t7">Lorem ipsum dolor sit amet, consectetur</tspan></text><text transform="translate(5.34 70.59)"><tspan x="0" y="0" class="t7">adipiscing elit, sed do eiusmod tempor</tspan></text><text transform="translate(5.34 83.02)"><tspan x="0" y="0" class="t7">incididunt ut labore et dolore magna.</tspan></text></svg>';
    }
  });

  tippy('span.vp-c-help', {
    content(help) {
      return '<svg version="1.2" viewBox="0 0 195 105" width="195px" height="105px"><style>tspan{white-space:pre}.t7,.t8{fill:var(--muted-color);font-size:10px;font-family:var(--font-family)}.t8{font-size:11px;fill:var(--primary)}</style><path fill="var(--card-background-color)" d="M.2.2h225v107H.2z"/><path fill="var(--muted-color)" d="M7.2 10.6c0-1.9 1.6-3.5 3.5-3.5h17c1.9 0 3.5 1.6 3.5 3.5v17c0 2-1.6 3.5-3.5 3.5h-17c-1.9 0-3.5-1.5-3.5-3.5z"/><path fill="var(--primary)" d="M35.5 11.6h90v6h-90zm0 9h90v6h-90z"/><path fill="var(--muted-color)" d="M5.2 83.1c114.1.6 215 0 215 0"/><text transform="translate(8.72 97.5)"><tspan x="0" y="0" class="t8">R</tspan></text><text transform="translate(40.66 97.5)"><tspan x="0" y="0" class="t8">E</tspan></text><text transform="translate(73.45 97.5)"><tspan x="0" y="0" class="t8">Y</tspan></text><text transform="translate(5.17 46.81)"><tspan x="0" y="0" class="t7">Lorem ipsum dolor sit amet, consectetur</tspan></text><text transform="translate(5.17 59.24)"><tspan x="0" y="0" class="t7">adipiscing elit, sed do eiusmod tempor</tspan></text><text transform="translate(5.17 71.67)"><tspan x="0" y="0" class="t7">incididunt ut labore et dolore magna.</tspan></text><path fill="var(--muted-color)" d="M155.8 11.6h29.7v6h-29.7zm0 9h29.7v6h-29.7z"/></svg>';
    }
  });

  tippy('span.vp-f-help', {
    content(help) {
      return '<svg version="1.2" viewBox="0 0 195 145" width="195px" height="145px"><style>tspan{white-space:pre}.s3{fill:var(--primary)}.t6,.t7{font-size:11px;fill:var(--primary-inverse);font-family:var(--font-family)}.t7{font-size:10px;fill:var(--muted-color)}</style><path fill="var(--card-background-color)" d="M.2.1h225v145H.2z"/><path fill="var(--muted-color)" d="M8.2 16.6c0-3.8 3-6.8 6.7-6.8h18.5c3.7 0 6.8 3 6.8 6.8V35c0 3.7-3.1 6.8-6.8 6.8H14.9c-3.7 0-6.7-3.1-6.7-6.8z"/><path fill="var(--primary)" d="M43.5 13.8h90v6h-90zm0 9h90v6h-90z"/><path d="M8.7 114.1h45v24h-45z" class="s3"/><path fill="var(--muted-color)" d="M43.5 31.8h90v6h-90zm-38.3 75c114.1.6 215 0 215 0"/><text transform="translate(14.08 130.35)"><tspan x="0" y="0" class="t6">Reply</tspan></text><path d="M62 113.9h45v24H62z" class="s3"/><text transform="translate(67.3 130.35)"><tspan x="0" y="0" class="t6">Edit</tspan></text><path d="M115 113.9h45v24h-45z" class="s3"/><text transform="translate(120.18 130.35)"><tspan x="0" y="0" class="t6">Yarn</tspan></text><text transform="translate(7.72 64.62)"><tspan x="0" y="0" class="t7">Lorem ipsum dolor sit amet, consectetur</tspan></text><text transform="translate(7.72 77.05)"><tspan x="0" y="0" class="t7">adipiscing elit, sed do eiusmod tempor</tspan></text><text transform="translate(7.72 89.48)"><tspan x="0" y="0" class="t7">incididunt ut labore et dolore magna.</tspan></text></svg>';
    }
  });
}
document.addEventListener("DOMContentLoaded", createTippyInstances);
document.addEventListener("htmx:afterSettle", createTippyInstances);

window.onload = function () {
  localStorage.setItem("isPost", "false");

  if (document.getElementById('postbox') != null) {
    usedBookmarklet()
    var text = localStorage.getItem('text');
    if (text && u("textarea#text").length > 0) {
      u("textarea#text").first().value = text;
      u("#newPost").attr("open", "");
      u("textarea#text").first().focus();
    }
  }
}
