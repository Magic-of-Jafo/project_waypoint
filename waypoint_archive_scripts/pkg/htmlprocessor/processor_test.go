package htmlprocessor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

const sampleHTMLPage = `
<?xml version="1.0" encoding="utf-8"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
<head><meta charset="utf-8">
<link rel="shortcut icon" href="../cafe.ico" />
<title>The Magic Cafe Forums - Dai Vernon and Houdini</title>
<style type="text/css">
body {font-size:15px;}
</style>
<!--The Magic Café - Visit us to discuss with others the wonderful world of magic and illusion.-->
<meta name="description" content="The Magic Café - Visit us to discuss with others the wonderful world of magic and illusion.">
<meta name="keywords" content="magic,forum,forums,community,street,steve,brooks,magicians,wizards,tricks,illusion,illusions,juggling,clowns,discussion,chat,message board">
<meta name="Author" content="Steve Brooks">
<meta name="Copyright" content="August 2001">
<meta http-equiv="content-language" content="English">
<meta name="MSSmartTagsPreventParsing" content="TRUE">

<link rel="stylesheet" type="text/css" href="cafe.css" />
<link type="text/css" href="jquery.toastmessage-min.css" rel="stylesheet"/>
<script type="text/javascript" src="jquery-latest.min.js"></script>
<script type="text/javascript" src="jquery.toastmessage-min.js"></script>
<script type="text/javascript">
	function toast(type,msg) {
		$().toastmessage('showToast', {
			text     : msg,
			sticky   : false,
			position : 'middle-center',
			type     : type,
			closeText: '',
			close    : function () {
				console.log("toast is closed ...");
			}
		});
		return true;
	}
	function stickytoast(type,msg) {
		$().toastmessage('showToast', {
			text     : msg,
			sticky   : true,
			position : 'middle-center',
			type     : type,
			closeText: '',
			close    : function () {
				console.log("toast is closed ...");
			}
		});
		return true;
	}
</script>
</head>
<body bgcolor="#000000"><div id="container">
<table class="normalnb" cellpadding="4" cellspacing="0">
	<tr>                    
		<td class="normalnb c w50">
			<div class="c">
				<a href="index.php"><img class="nb vam" src="images/header2.gif" alt="The Magic Caf" title="The Magic Caf" /></a>
			</div>
		<td class="b c">
			<div class="c">
				[ <a href="faq.php">F.A.Q.</a> ]
				<br />[<a href="donate.php"> Magic Caf&eacute; Donations </a>]
			</div>
			<div class="c">
				<table class="nb tc" cellpadding="0" cellspacing="8">
					<tr>
						<td class="b r">
							<form action="login.php" method="post" style="display:inline;">
							Username: <input type="text" name="user" size="15" maxlength="40" /><br />
							Password: <input type="password" name="passwd" size="15" /><br />
							<input class="submit" type="submit" name="submit" value="Log In" />
							</form>
						</td>
					</tr>
					<tr>
						<td class="c">[ <a href="sendpassword.php">Lost Password</a> ]<br />&nbsp;&nbsp;[ <a href="forgotusername.php">Forgot Username</a> ]</td>
					</tr>
				</table>
			</div>
		</td>
	</tr>
</table>
<table class="normal" cellpadding="4" cellspacing="1"> <!-- First table.normal (breadcrumbs) -->
	<tr>
		<td class="normal bgc1" colspan="2">
			<table class="nb w100" cellpadding="0" cellspacing="0">
				<tr>
					<td class="w99 mltext"><a href="index.php">The Magic Cafe Forum Index</a> &raquo; &raquo; <a href="viewforum.php?forum=66">What happened, was this...</a> &raquo; &raquo; Dai Vernon and Houdini<span class="midtext">&nbsp;(0&nbsp;Likes)</span></td>
					<td class="w2 vam"><a href="printtopic.php?topic=19618&amp;forum=66" target="_blank"><img class="nb" src="images/print.gif" alt="Printer Friendly Version" title="Printer Friendly Version" hspace="4" /></a></td>
				</tr>
			</table>
		</td>
	</tr>
</table>
<br />
<table class="normal" cellpadding="4" cellspacing="1"> <!-- Second table.normal (posts) -->
	<tr> <!-- Pagination Row, should not be counted as a post -->
		<td class="normal bgc2 b midtext" colspan="11">
			&nbsp;Go to page [<a href="viewtopic.php?topic=19618&amp;start=0" title="Previous Page" alt="Previous Page">Previous</a>]&nbsp;&nbsp;<a href="viewtopic.php?topic=19618&amp;start=0" title="Page 1" alt="Page 1">1</a>~<span class="on_page">2</span>
		</td>
	</tr>
	<tr> <!-- Post 1 -->
		<td class="normal bgc1 c w13 vat">
			<strong>Maxim</strong><br />
			<a href="bb_profile.php?mode=view&amp;user=5267"><img class="nb" src="images/avatars/nopic.gif" vspace="3" alt="View Profile" title="View Profile" /></a><br /><span class="smalltext">
			<strong>Regular user</strong><br />
			London<br />
			113 Posts</span><br />
			<a href="bb_profile.php?mode=view&amp;user=5267"><img class="nb vab" src="images/profile.gif" alt="Profile of Maxim" title="Profile of Maxim" /></a>
		</td>
		<td class="normal bgc1 vat w90">
			<div class="vt1 liketext">
				<div class="like_left">
					<span class="b">
					<a name="0"></a><img class="nb vam" src="images/posticon.gif" alt="Post Icon" title="Post Icon" /> 
					Posted: Jan 23, 2003 02:45 pm </span>&nbsp;&nbsp;
				</div>
				<div class="like_right"><img class="vab" src="images/likes.gif" alt="There are no likes for this post." title="There are no  likes for this post." /><span id="p_175716">0</span></div>
			</div>
			<div class="w100">

<!-- POST TEXT -->

Hey Dave do you know the web link to RK?<br><br>You still gotta hand it do Dai Vernon, he may have not been the nicest chap after all, but at least he spoke his mind. Not like the awful 'sucking up' that goes on today in magic. One word against a famous magician and you're considered a traitor!<br><br>Maxim.
<!-- END POST TEXT -->

			</div>
		</td>
	</tr>
	<tr> <!-- Post 2 -->
		<td class="normal bgc1 c w13 vat">
			<strong>rkrahlmann</strong><br />
			<a href="bb_profile.php?mode=view&amp;user=3416"><img class="nb" src="images/avatars/nopic.gif" vspace="3" alt="View Profile" title="View Profile" /></a><br /><span class="smalltext">
			<strong>Regular user</strong><br />
			168 Posts</span><br />
			<a href="bb_profile.php?mode=view&amp;user=3416"><img class="nb vab" src="images/profile.gif" alt="Profile of rkrahlmann" title="Profile of rkrahlmann" /></a>
		</td>
		<td class="normal bgc1 vat w90">
			<div class="vt1 liketext">
				<div class="like_left">
					<span class="b">
					<a name="1"></a><img class="nb vam" src="images/posticon.gif" alt="Post Icon" title="Post Icon" /> 
					Posted: Jan 23, 2003 09:59 pm </span>&nbsp;&nbsp;
				</div>
				<div class="like_right"><img class="vab" src="images/likes.gif" alt="There are no likes for this post." title="There are no  likes for this post." /><span id="p_176021">0</span></div>
			</div>
			<div class="w100">

<!-- POST TEXT -->

If you have access to a library with back issues of The New Yorker, check out the April 5th, 1993 issue featuring a profile of Ricky Jay. There's a number of inches devoted to the Professor. Particularly of note is his response to Jay's request for insturction and insight. I can't print it here.<br>A friend of Vernon's is quoted "I wouldn't have taken a million dollars not to have known him. But I'd give a million not to know another one like him."
<!-- END POST TEXT -->

			</div>
		</td>
	</tr>
	<tr> <!-- Post 3 -->
		<td class="normal bgc1 c w13 vat">
			<strong>Dave Egleston</strong><br />
			<a href="bb_profile.php?mode=view&amp;user=2367"><img class="nb" src="images/avatars/2367_Picture_007.jpg" vspace="3" alt="View Profile" title="View Profile" /></a><br /><span class="smalltext">
			<strong>Special user</strong><br />
			Ceres, Ca<br />
			632 Posts</span><br />
			<a href="bb_profile.php?mode=view&amp;user=2367"><img class="nb vab" src="images/profile.gif" alt="Profile of Dave Egleston" title="Profile of Dave Egleston" /></a>
		</td>
		<td class="normal bgc1 vat w90">
			<div class="vt1 liketext">
				<div class="like_left">
					<span class="b">
					<a name="2"></a><img class="nb vam" src="images/posticon.gif" alt="Post Icon" title="Post Icon" /> 
					Posted: Jan 25, 2003 03:46 am </span>&nbsp;&nbsp;
				</div>
				<div class="like_right"><img class="vab" src="images/likes.gif" alt="There are no likes for this post." title="There are no  likes for this post." /><span id="p_176968">0</span></div>
			</div>
			<div class="w100">

<!-- POST TEXT -->

Maxim,<br>It's the Genii Magazine website<br><br><!-- BBCode u1 Start 1 --><a href="http://www.geniimagazine.com" target="_blank">http://www.geniimagazine.com</a><!-- BBCode url End --><br><br>It's a pretty good website - a little more aggresive than this site - They don't tolerate "trolling" type questions. ie: "Who's your favorite - Marlo or Vernon"<br><br>Dave
<!-- END POST TEXT -->

			</div>
		</td>
	</tr>
	<tr> <!-- Post 4 -->
		<td class="normal bgc1 c w13 vat">
			<strong>Bill Hallahan</strong><br />
			<a href="bb_profile.php?mode=view&amp;user=1419"><img class="nb" src="images/avatars/1419_bill_hallahan_avatar.jpg" vspace="3" alt="View Profile" title="View Profile" /></a><br /><span class="smalltext">
			<strong>Inner circle</strong><br />
			New Hampshire<br />
			3231 Posts</span><br />
			<a href="bb_profile.php?mode=view&amp;user=1419"><img class="nb vab" src="images/profile.gif" alt="Profile of Bill Hallahan" title="Profile of Bill Hallahan" /></a>
		</td>
		<td class="normal bgc1 vat w90">
			<div class="vt1 liketext">
				<div class="like_left">
					<span class="b">
					<a name="3"></a><img class="nb vam" src="images/posticon.gif" alt="Post Icon" title="Post Icon" /> 
					Posted: Aug 13, 2003 08:18 pm </span>&nbsp;&nbsp;
				</div>
				<div class="like_right"><img class="vab" src="images/likes.gif" alt="There are no likes for this post." title="There are no  likes for this post." /><span id="p_334279">0</span></div>
			</div>
			<div class="w100">

<!-- POST TEXT -->

<div class="quote">Quote:<div class="quote_inner"><blockquote>There was only one conjuror that he spoke of negatively, and that was Harry Houdini.</blockquote></div></div><br>I'm not sure that Dai Vernon didn't like Houdini. It's very clear he didn't think much of his magic.<br><br>Clearly he also spoke negatively about Michael Ammar, but I can assure you that there was a great deal of respect in both directions between these two men. Michael Ammar still says he keeps Dai Vernon on a pedestal.<br><br>I've met Mr. Ammar and seen him both perform and lecture at a local SAM assembly meeting. He was engaging and highly entertaining. I like his laugh. I can understand someone not being a fan of his style. I cannot understand why anyone would enjoy seeing him get put down. He is not just one of the nicest magician's I have ever met; he is one of the nicest people I have ever met.<br><br>Finally, I admire Dai Vernon's students for realizing the worth of putting up with a magic "boot camp" environment. At the same time, I don't think that teaching style is appropriate for magic (although people can die on stage!)
<div class="vt2">
Humans make life so interesting.  Do you know that in a universe so full of wonders, they have managed to create boredom.   Quite astonishing.<br />  - The character of 'Death'  in the movie "Hogswatch"

<!-- END POST TEXT -->

			</div>
		</td>
	</tr>
	<tr> <!-- Post 5 -->
		<td class="normal bgc1 c w13 vat">
			<strong>ursusminor</strong><br />
			<a href="bb_profile.php?mode=view&amp;user=10418"><img class="nb" src="images/avatars/nopic.gif" vspace="3" alt="View Profile" title="View Profile" /></a><br /><span class="smalltext">
			<strong>Elite user</strong><br />
			Norway<br />
			443 Posts</span><br />
			<a href="bb_profile.php?mode=view&amp;user=10418"><img class="nb vab" src="images/profile.gif" alt="Profile of ursusminor" title="Profile of ursusminor" /></a>
		</td>
		<td class="normal bgc1 vat w90">
			<div class="vt1 liketext">
				<div class="like_left">
					<span class="b">
					<a name="4"></a><img class="nb vam" src="images/posticon.gif" alt="Post Icon" title="Post Icon" /> 
					Posted: Feb 25, 2004 03:09 pm </span>&nbsp;&nbsp;
				</div>
				<div class="like_right"><img class="vab" src="images/likes.gif" alt="There are no likes for this post." title="There are no  likes for this post." /><span id="p_3584589">0</span></div>
			</div>
			<div class="w100">

<!-- POST TEXT -->

<div class="quote">Quote:<div class="quote_inner"><blockquote>On 2003-01-21 20:12, Dave Egleston wrote:<br>You have to watch it to appreciate the disdain he holds for Ammar - Mr Vernon had no patience for Ammar's lack of preparedness and considered some of Ammar's questions  "inappropriate?" from a professional magician.<br><br>As they say - "Worth the price of the tape"<br><br>Dave<br></blockquote></div></div><br><br>Unfortunately that particular scene is "missing" from the reissue of the Revelations-tapes...<br><br>Bjørn</div>
<div class="vt2">
"Men occasionally stumble over the truth, but most of them<br />pick themselves up and hurry off as if nothing happened."<br />  - Winston Churchill"

<!-- END POST TEXT -->

			</div>
		</td>
	</tr>
	<tr> <!-- Footer Row 1, should not be counted -->
		<td class="normal bgc2 mltext" colspan="2"><a href="index.php">The Magic Cafe Forum Index</a> &raquo; &raquo; <a href="viewforum.php?forum=66">What happened, was this...</a> &raquo; &raquo; Dai Vernon and Houdini<span class="midtext">&nbsp;(0&nbsp;Likes)</span></td>
	</tr>
	<tr> <!-- Footer Row 2 (Pagination), should not be counted -->
		<td class="normal bgc2 b midtext" colspan="11">			&nbsp;Go to page [<a href="viewtopic.php?topic=19618&amp;forum=66&amp;start=0" title="Previous Page" alt="Previous Page">Previous</a>]&nbsp;&nbsp;<a href="viewtopic.php?topic=19618&amp;forum=66&amp;start=0" title="Page 1" alt="Page 1">1</a>~<span class="on_page">2</span>		</td>
	</tr>
</table>
<table class="normalnb c b" cellpadding="10" cellspacing="0"><tr><td class="c">[ <a href="#">Top of Page</a> ]</td></tr></table>
<table class="normal c bgc2"  cellpadding="2" cellspacing="0">
	<tr>
		<td class="c smalltext">
			All content &amp; postings Copyright &copy; 2001-2025 Steve Brooks. All Rights Reserved.<br />
This page was created in 0.01 seconds requiring 5 database queries.
		</td>
	</tr>
</table>
<table class="normalnb c" cellpadding="2" cellspacing="0">
	<tr>
		<td class="c smalltext yellow">
			The views and comments expressed on <span class="green b">The Magic Caf&eacute;</span><br />
							are not necessarily those of <span class="green b">The Magic Caf&eacute;,</span> Steve Brooks, or
			Steve Brooks Magic.<br />
			<a class="b" href="http://themagiccafe.com/privacy.html">&gt; Privacy Statement &lt;</a><br /><br />
			<img src="images/smiles/rotfl.gif" alt="ROTFL" title="ROTFL" style="padding-right:25px;vertical-align:middle;" />
			<img src="images/bserved.gif" alt="Billions and billions served!" title="Billions and billions served!" style="vertical-align:middle;" />
			<img src="images/smiles/rotfl.gif" alt="ROTFL" title="ROTFL" style="padding-left:25px;vertical-align:middle;" /><br />
		</td>
	</tr>
</table>
</div>
</body>
</html>
`

func TestLoadHTMLPage(t *testing.T) {
	// Create a temporary test HTML file using a simpler HTML for basic loading test
	tempDir := t.TempDir()
	simpleHTML := `<html><body><div class="post">Test post content</div></body></html>`
	testFilePath := filepath.Join(tempDir, "simple.html")

	err := os.WriteFile(testFilePath, []byte(simpleHTML), 0644)
	if err != nil {
		t.Fatalf("Failed to create simple test HTML file: %v", err)
	}

	// Test loading the HTML file
	page, err := LoadHTMLPage(testFilePath)
	if err != nil {
		t.Fatalf("LoadHTMLPage failed for simple HTML: %v", err)
	}

	// Verify the page was loaded correctly
	if page.FilePath != testFilePath {
		t.Errorf("Expected file path %s, got %s", testFilePath, page.FilePath)
	}
	if page.Content == nil {
		t.Error("Expected non-nil Content document for simple HTML")
	}
	content := page.Content.Find("div.post").Text()
	if content != "Test post content" {
		t.Errorf("Expected content 'Test post content', got '%s' for simple HTML", content)
	}
}

func TestLoadHTMLPage_NonexistentFile(t *testing.T) {
	_, err := LoadHTMLPage("nonexistent.html")
	if err == nil {
		t.Error("Expected error when loading nonexistent file")
	}
}

func TestGetPostBlocks_SamplePage(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(sampleHTMLPage))
	if err != nil {
		t.Fatalf("Failed to parse sampleHTMLPage: %v", err)
	}

	htmlPage := &HTMLPage{
		FilePath: "sample/page.html",
		Content:  doc,
	}

	blocks, err := htmlPage.GetPostBlocks()
	if err != nil {
		t.Fatalf("GetPostBlocks failed for sample page: %v", err)
	}

	expectedPostCount := 5
	if len(blocks) != expectedPostCount {
		t.Errorf("Expected %d post blocks, got %d", expectedPostCount, len(blocks))
	}

	// Optional: Further checks on individual blocks if needed
	// For example, check if each block contains the expected structure or some specific text
	for i, block := range blocks {
		userCell := block.Selection.Find("td.normal.bgc1.c.w13.vat")
		if userCell.Length() == 0 {
			t.Errorf("Post block %d missing user cell", i)
		}
		postContentCell := block.Selection.Find("td.normal.bgc1.vat.w90")
		if postContentCell.Length() == 0 {
			t.Errorf("Post block %d missing post content cell", i)
		}
		// Example of checking some text within a post - requires knowing expected text
		// if i == 0 && !strings.Contains(postContentCell.Find("div.w100").Text(), "Hey Dave") {
		// 	t.Errorf("Post block 0 does not seem to contain expected text 'Hey Dave'")
		// }
	}
}

func TestGetPostBlocks_EmptyPage(t *testing.T) {
	emptyHTML := `<html><body><div id="container"><table class="normal"></table></div></body></html>`
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(emptyHTML))
	htmlPage := &HTMLPage{FilePath: "empty.html", Content: doc}

	blocks, err := htmlPage.GetPostBlocks()
	if err != nil {
		t.Fatalf("GetPostBlocks failed for empty page: %v", err)
	}
	if len(blocks) != 0 {
		t.Errorf("Expected 0 post blocks for empty page, got %d", len(blocks))
	}
}

func TestGetPostBlocks_NoMatchingTable(t *testing.T) {
	noMatchingTableHTML := `<html><body><div id="container"><table class="other_table"><tr><td>No posts here</td></tr></table></div></body></html>`
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(noMatchingTableHTML))
	htmlPage := &HTMLPage{FilePath: "no_matching_table.html", Content: doc}

	blocks, err := htmlPage.GetPostBlocks()
	if err != nil {
		t.Fatalf("GetPostBlocks failed for page with no matching table: %v", err)
	}
	if len(blocks) != 0 {
		t.Errorf("Expected 0 post blocks for page with no matching table, got %d", len(blocks))
	}
}

func TestGetPostBlocks_NoMatchingRowsInTable(t *testing.T) {
	noMatchingRowsHTML := `
	<html><body><div id="container">
		<table class="normal"> <!-- Breadcrumb Table --><tr><td>Nav</td></tr></table>
		<table class="normal"> <!-- Post Table, but no valid post rows -->
			<tr><td class="normal bgc2 b midtext">Pagination</td></tr>
			<tr><td>Not a post row</td></tr>
		</table>
	</div></body></html>`
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(noMatchingRowsHTML))
	htmlPage := &HTMLPage{FilePath: "no_matching_rows.html", Content: doc}

	blocks, err := htmlPage.GetPostBlocks()
	if err != nil {
		t.Fatalf("GetPostBlocks failed for page with no matching rows: %v", err)
	}
	if len(blocks) != 0 {
		t.Errorf("Expected 0 post blocks for page with no matching rows, got %d", len(blocks))
	}
}
