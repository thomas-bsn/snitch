package httpapi

import (
	"reflect"
	"testing"
)

func TestFolderNameTokens(t *testing.T) {
	for _, tc := range []struct{ name, want string }{
		{"Among Us 3d Demo", "among us"},
		{"Bread.and.fred-Goldberg", "bread and fred"},
		{"Dark Souls Remastered [FitGirl Repack]", "dark souls remastered"},
		{"Call of Duty Black Ops 2", "call of duty black ops 2"},
		{"Counter-Strike: Global Offensive", "counter strike global offensive"},
		{"Fall Guys v1.2.3", "fall guys"},
		{"Content", "content"},
	} {
		if got := folderLogoSearchName(tc.name); got != tc.want {
			t.Errorf("%q: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestFolderLogoKeyIgnoresQualifiers(t *testing.T) {
	if folderLogoKey("Fall Guys 3d Demo") != folderLogoKey("fall guys") {
		t.Fatal("a build qualifier must not change the cache key")
	}
	if folderLogoKey("Bread.and.fred-Goldberg") != folderLogoKey("Bread and Fred") {
		t.Fatal("a scene tag must not change the cache key")
	}
}

func TestChooseFolderGame(t *testing.T) {
	item := func(id int64, name, metascore string) steamSearchItem {
		return steamSearchItem{Type: "app", ID: id, Name: name, Metascore: metascore}
	}
	for _, tc := range []struct {
		name   string
		folder string
		items  []steamSearchItem
		want   int64
	}{
		{
			name:   "the exact title wins over a longer prefix",
			folder: "Among Us 3d Demo",
			items:  []steamSearchItem{item(1, "The Wolf Among Us", ""), item(945360, "Among Us", ""), item(3168600, "Among Us 3D", "")},
			want:   945360,
		},
		{
			name:   "a scene release tag does not prevent the match",
			folder: "Bread.and.fred-Goldberg",
			items:  []steamSearchItem{item(1607680, "Bread & Fred", "80"), item(1, "Bread & Fred Demo", ""), item(2, "Bread & Fred Soundtrack", "")},
			want:   1607680,
		},
		{
			name:   "roman numerals and digits are the same title",
			folder: "Call of Duty Black Ops 2",
			items:  []steamSearchItem{item(202970, "Call of Duty®: Black Ops II", "83"), item(219103, "Call of Duty®: Black Ops II - Apocalypse", "")},
			want:   202970,
		},
		{
			name:   "a subtitle is accepted, and the main game beats a spinoff",
			folder: "Earth Defense Force 4.1",
			items:  []steamSearchItem{item(574200, "EARTH DEFENSE FORCE 4.1 WINGDIVER THE SHOOTER", ""), item(410320, "EARTH DEFENSE FORCE 4.1 The Shadow of New Despair", "79")},
			want:   410320,
		},
		{
			name:   "an edition suffix on the folder is accepted",
			folder: "Dark Souls Remastered",
			items:  []steamSearchItem{item(570940, "DARK SOULS™: REMASTERED", "84")},
			want:   570940,
		},
		{
			name:   "a generic folder is never grown into a game",
			folder: "Content",
			items:  []steamSearchItem{item(1, "Content Warning", "80")},
			want:   0,
		},
		{
			name:   "a system folder is refused even on an exact title",
			folder: "Clips",
			items:  []steamSearchItem{item(1, "Clips", "")},
			want:   0,
		},
		{
			name:   "a missing game stays missing",
			folder: "Fall Guys",
			items:  nil,
			want:   0,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := chooseFolderGame(folderNameTokens(tc.folder), tc.items)
			var id int64
			if got != nil {
				id = got.ID
			}
			if id != tc.want {
				t.Fatalf("got app %d, want %d", id, tc.want)
			}
		})
	}
}

func TestFolderNameTokensKeepWords(t *testing.T) {
	want := []string{"i", "am", "fish"}
	if got := folderNameTokens("I Am Fish"); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if got := folderNameTokens("V Rising"); !reflect.DeepEqual(got, []string{"v", "rising"}) {
		t.Fatalf("got %v, want [v rising]", got)
	}
	if got := folderNameTokens("Grand Theft Auto V"); !reflect.DeepEqual(got, []string{"grand", "theft", "auto", "5"}) {
		t.Fatalf("got %v, want [grand theft auto 5]", got)
	}
}
