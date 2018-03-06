package formatter

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/stringid"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
)

func TestImageContext(t *testing.T) {
	imageID := stringid.GenerateRandomID()
	unix := time.Now().Unix()

	var ctx imageContext
	cases := []struct {
		imageCtx imageContext
		expValue string
		call     func() string
	}{
		{imageContext{
			i:     types.ImageSummary{ID: imageID},
			trunc: true,
		}, stringid.TruncateID(imageID), ctx.ID},
		{imageContext{
			i:     types.ImageSummary{ID: imageID},
			trunc: false,
		}, imageID, ctx.ID},
		{imageContext{
			i:     types.ImageSummary{Size: 10, VirtualSize: 10},
			trunc: true,
		}, "10B", ctx.Size},
		{imageContext{
			i:     types.ImageSummary{Created: unix},
			trunc: true,
		}, time.Unix(unix, 0).String(), ctx.CreatedAt},
		// FIXME
		// {imageContext{
		// 	i:     types.ImageSummary{Created: unix},
		// 	trunc: true,
		// }, units.HumanDuration(time.Unix(unix, 0)), createdSinceHeader, ctx.CreatedSince},
		{imageContext{
			i:    types.ImageSummary{},
			repo: "busybox",
		}, "busybox", ctx.Repository},
		{imageContext{
			i:   types.ImageSummary{},
			tag: "latest",
		}, "latest", ctx.Tag},
		{imageContext{
			i:      types.ImageSummary{},
			digest: "sha256:d149ab53f8718e987c3a3024bb8aa0e2caadf6c0328f1d9d850b2a2a67f2819a",
		}, "sha256:d149ab53f8718e987c3a3024bb8aa0e2caadf6c0328f1d9d850b2a2a67f2819a", ctx.Digest},
		{
			imageContext{
				i: types.ImageSummary{Containers: 10},
			}, "10", ctx.Containers,
		},
		{
			imageContext{
				i: types.ImageSummary{VirtualSize: 10000},
			}, "10kB", ctx.VirtualSize,
		},
		{
			imageContext{
				i: types.ImageSummary{SharedSize: 10000},
			}, "10kB", ctx.SharedSize,
		},
		{
			imageContext{
				i: types.ImageSummary{SharedSize: 5000, VirtualSize: 20000},
			}, "15kB", ctx.UniqueSize,
		},
	}

	for _, c := range cases {
		ctx = c.imageCtx
		v := c.call()
		if strings.Contains(v, ",") {
			compareMultipleValues(t, v, c.expValue)
		} else {
			assert.Check(t, is.Equal(c.expValue, v))
		}
	}
}

func TestImageContextWrite(t *testing.T) {
	unixTime := time.Now().AddDate(0, 0, -1).Unix()
	expectedTime := time.Unix(unixTime, 0).String()

	cases := []struct {
		context  ImageContext
		expected string
	}{
		// Errors
		{
			ImageContext{
				Context: Context{
					Format: "{{InvalidFunction}}",
				},
			},
			`Template parsing error: template: :1: function "InvalidFunction" not defined
`,
		},
		{
			ImageContext{
				Context: Context{
					Format: "{{nil}}",
				},
			},
			`Template parsing error: template: :1:2: executing "" at <nil>: nil is not a command
`,
		},
		// Table Format
		{
			ImageContext{
				Context: Context{
					Format: NewImageFormat("table", false, false),
				},
			},
			`REPOSITORY          TAG                 IMAGE ID            CREATED             SIZE
image               tag1                imageID1            24 hours ago        0B
image               tag2                imageID2            24 hours ago        0B
<none>              <none>              imageID3            24 hours ago        0B
`,
		},
		{
			ImageContext{
				Context: Context{
					Format: NewImageFormat("table {{.Repository}}", false, false),
				},
			},
			"REPOSITORY\nimage\nimage\n<none>\n",
		},
		{
			ImageContext{
				Context: Context{
					Format: NewImageFormat("table {{.Repository}}", false, true),
				},
				Digest: true,
			},
			`REPOSITORY          DIGEST
image               sha256:cbbf2f9a99b47fc460d422812b6a5adff7dfee951d8fa2e4a98caa0382cfbdbf
image               <none>
<none>              <none>
`,
		},
		{
			ImageContext{
				Context: Context{
					Format: NewImageFormat("table {{.Repository}}", true, false),
				},
			},
			"REPOSITORY\nimage\nimage\n<none>\n",
		},
		{
			ImageContext{
				Context: Context{
					Format: NewImageFormat("table {{.Digest}}", true, false),
				},
			},
			"DIGEST\nsha256:cbbf2f9a99b47fc460d422812b6a5adff7dfee951d8fa2e4a98caa0382cfbdbf\n<none>\n<none>\n",
		},
		{
			ImageContext{
				Context: Context{
					Format: NewImageFormat("table", true, false),
				},
			},
			"imageID1\nimageID2\nimageID3\n",
		},
		{
			ImageContext{
				Context: Context{
					Format: NewImageFormat("table", false, true),
				},
				Digest: true,
			},
			`REPOSITORY          TAG                 DIGEST                                                                    IMAGE ID            CREATED             SIZE
image               tag1                sha256:cbbf2f9a99b47fc460d422812b6a5adff7dfee951d8fa2e4a98caa0382cfbdbf   imageID1            24 hours ago        0B
image               tag2                <none>                                                                    imageID2            24 hours ago        0B
<none>              <none>              <none>                                                                    imageID3            24 hours ago        0B
`,
		},
		{
			ImageContext{
				Context: Context{
					Format: NewImageFormat("table", true, true),
				},
				Digest: true,
			},
			"imageID1\nimageID2\nimageID3\n",
		},
		// Raw Format
		{
			ImageContext{
				Context: Context{
					Format: NewImageFormat("raw", false, false),
				},
			},
			fmt.Sprintf(`repository: image
tag: tag1
image_id: imageID1
created_at: %s
virtual_size: 0B

repository: image
tag: tag2
image_id: imageID2
created_at: %s
virtual_size: 0B

repository: <none>
tag: <none>
image_id: imageID3
created_at: %s
virtual_size: 0B

`, expectedTime, expectedTime, expectedTime),
		},
		{
			ImageContext{
				Context: Context{
					Format: NewImageFormat("raw", false, true),
				},
				Digest: true,
			},
			fmt.Sprintf(`repository: image
tag: tag1
digest: sha256:cbbf2f9a99b47fc460d422812b6a5adff7dfee951d8fa2e4a98caa0382cfbdbf
image_id: imageID1
created_at: %s
virtual_size: 0B

repository: image
tag: tag2
digest: <none>
image_id: imageID2
created_at: %s
virtual_size: 0B

repository: <none>
tag: <none>
digest: <none>
image_id: imageID3
created_at: %s
virtual_size: 0B

`, expectedTime, expectedTime, expectedTime),
		},
		{
			ImageContext{
				Context: Context{
					Format: NewImageFormat("raw", true, false),
				},
			},
			`image_id: imageID1
image_id: imageID2
image_id: imageID3
`,
		},
		// Custom Format
		{
			ImageContext{
				Context: Context{
					Format: NewImageFormat("{{.Repository}}", false, false),
				},
			},
			"image\nimage\n<none>\n",
		},
		{
			ImageContext{
				Context: Context{
					Format: NewImageFormat("{{.Repository}}", false, true),
				},
				Digest: true,
			},
			"image\nimage\n<none>\n",
		},
	}

	for _, testcase := range cases {
		images := []types.ImageSummary{
			{ID: "imageID1", RepoTags: []string{"image:tag1"}, RepoDigests: []string{"image@sha256:cbbf2f9a99b47fc460d422812b6a5adff7dfee951d8fa2e4a98caa0382cfbdbf"}, Created: unixTime},
			{ID: "imageID2", RepoTags: []string{"image:tag2"}, Created: unixTime},
			{ID: "imageID3", RepoTags: []string{"<none>:<none>"}, RepoDigests: []string{"<none>@<none>"}, Created: unixTime},
		}
		out := bytes.NewBufferString("")
		testcase.context.Output = out
		err := ImageWrite(testcase.context, images)
		if err != nil {
			assert.Error(t, err, testcase.expected)
		} else {
			assert.Check(t, is.Equal(testcase.expected, out.String()))
		}
	}
}

func TestImageContextWriteWithNoImage(t *testing.T) {
	out := bytes.NewBufferString("")
	images := []types.ImageSummary{}

	contexts := []struct {
		context  ImageContext
		expected string
	}{
		{
			ImageContext{
				Context: Context{
					Format: NewImageFormat("{{.Repository}}", false, false),
					Output: out,
				},
			},
			"",
		},
		{
			ImageContext{
				Context: Context{
					Format: NewImageFormat("table {{.Repository}}", false, false),
					Output: out,
				},
			},
			"REPOSITORY\n",
		},
		{
			ImageContext{
				Context: Context{
					Format: NewImageFormat("{{.Repository}}", false, true),
					Output: out,
				},
			},
			"",
		},
		{
			ImageContext{
				Context: Context{
					Format: NewImageFormat("table {{.Repository}}", false, true),
					Output: out,
				},
			},
			"REPOSITORY          DIGEST\n",
		},
	}

	for _, context := range contexts {
		ImageWrite(context.context, images)
		assert.Check(t, is.Equal(context.expected, out.String()))
		// Clean buffer
		out.Reset()
	}
}
