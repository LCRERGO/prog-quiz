package main

import (
	"fmt"
	"io"
	"os"
)

var Genres []string = []string{
	"Blues",
	"Classic Rock",
	"Country",
	"Dance",
	"Disco",
	"Funk",
	"Grunge",
	"Hip-Hop",
	"Jazz",
	"Metal",
	"New Age",
	"Oldies",
	"Other",
	"Pop",
	"R&B",
	"Rap",
	"Reggae",
	"Rock",
	"Techno",
	"Industrial",
	"Alternative",
	"Ska",
	"Death Metal",
	"Pranks",
	"Soundtrack",
	"Euro-Techno",
	"Ambient",
	"Trip-Hop",
	"Vocal",
	"Jazz+Funk",
	"Fusion",
	"Trance",
	"Classical",
	"Instrumental",
	"Acid",
	"House",
	"Game",
	"Sound Clip",
	"Gospel",
	"Noise",
	"AlternRock",
	"Bass",
	"Soul",
	"Punk",
	"Space",
	"Meditative",
	"Instrumental Pop",
	"Instrumental Rock",
	"Ethnic",
	"Gothic",
	"Darkwave",
	"Techno-Industrial",
	"Electronic",
	"Pop-Folk",
	"Eurodance",
	"Dream",
	"Southern Rock",
	"Comedy",
	"Cult",
	"Gangsta",
	"Top 40",
	"Christian Rap",
	"Pop/Funk",
	"Jungle",
	"Native American",
	"Cabaret",
	"New Wave",
	"Psychadelic",
	"Rave",
	"Showtunes",
	"Trailer",
	"Lo-Fi",
	"Tribal",
	"Acid Punk",
	"Acid Jazz",
	"Polka",
	"Retro",
	"Musical",
	"Rock & Roll",
	"Hard Rock",
	"Folk",
	"Folk-Rock",
	"National Folk",
	"Swing",
	"Fast Fusion",
	"Bebob",
	"Latin",
	"Revival",
	"Celtic",
	"Bluegrass",
	"Avantgarde",
	"Gothic Rock",
	"Progressive Rock",
	"Psychedelic Rock",
	"Symphonic Rock",
	"Slow Rock",
	"Big Band",
	"Chorus",
	"Easy Listening",
	"Acoustic",
	"Humour",
	"Speech",
	"Chanson",
	"Opera",
	"Chamber Music",
	"Sonata",
	"Symphony",
	"Booty Bass",
	"Primus",
	"Porn Groove",
	"Satire",
	"Slow Jam",
	"Club",
	"Tango",
	"Samba",
	"Folklore",
	"Ballad",
	"Power Ballad",
	"Rhythmic Soul",
	"Freestyle",
	"Duet",
	"Punk Rock",
	"Drum Solo",
	"A capella",
	"Euro-House",
	"Dance Hall",
}

type ID3Tag struct {
	Song, Album     []byte
	Artist, Comment []byte
	Year            []byte
	Genre           byte
	TrackNumber     byte
}

func (i *ID3Tag) String() string {
	var genre string

	if int(i.Genre) >= 0 && int(i.Genre) < len(Genres) {
		genre = Genres[i.Genre]
	} else {
		genre = "Unknown"
	}

	return fmt.Sprintf(`
	{
	    Song: %s,
	    Artist: %s,
	    Album: %s,
	    Year: %s,
	    Comment: %s,
	    TrackNumber: %d
	    Genre: %s,
	}`,
		i.Song,
		i.Artist,
		i.Album,
		i.Year,
		i.Comment,
		i.TrackNumber,
		genre,
	)

}

func ReadTags(file *os.File) (*ID3Tag, error) {
	var err error
	var trackNumber byte

	buf := make([]byte, 128)
	_, err = file.Seek(-128, io.SeekEnd)
	if err != nil {
		return nil, err
	}

	_, err = file.Read(buf)
	if err != nil {
		return nil, err
	}

	if string(buf[:3]) != "TAG" {
		err = fmt.Errorf("Does not contain a id3 tag format")
		return nil, err
	}

	if buf[125] == 0x00 && buf[126] != 0x00 {
		trackNumber = buf[126]
	}

	return &ID3Tag{
		Song:        buf[3:33],
		Artist:      buf[33:63],
		Album:       buf[63:93],
		Year:        buf[93:97],
		Comment:     buf[97:127],
		TrackNumber: trackNumber,
		Genre:       buf[127],
	}, nil
}

func main() {
	var err error
	buf := make([]byte, 16)

	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s filename", os.Args[0])
		os.Exit(1)
	}

	file, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	tags, err := ReadTags(file)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Printf("%s\n", tags)

	os.Stdout.Write(buf)
	defer file.Close()
}
