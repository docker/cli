package formatter

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/stringid"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestImageContext(t *testing.T) {
	imageID := stringid.GenerateRandomID()
	unix := time.Now().Unix()
	zeroTime := int64(-62135596800)

	var ctx imageContext
	cases := []struct {
		imageCtx imageContext
		expValue string
		call     func() string
	}{
		{
			imageCtx: imageContext{i: types.ImageSummary{ID: imageID}, trunc: true},
			expValue: stringid.TruncateID(imageID),
			call:     ctx.ID,
		},
		{
			imageCtx: imageContext{i: types.ImageSummary{ID: imageID}, trunc: false},
			expValue: imageID,
			call:     ctx.ID,
		},
		{
			imageCtx: imageContext{i: types.ImageSummary{Size: 10, VirtualSize: 10}, trunc: true},
			expValue: "10B",
			call:     ctx.Size,
		},
		{
			imageCtx: imageContext{i: types.ImageSummary{Created: unix}, trunc: true},
			expValue: time.Unix(unix, 0).String(), call: ctx.CreatedAt,
		},
		// FIXME
		// {imageContext{
		// 	i:     types.ImageSummary{Created: unix},
		// 	trunc: true,
		// }, units.HumanDuration(time.Unix(unix, 0)), createdSinceHeader, ctx.CreatedSince},
		{
			imageCtx: imageContext{i: types.ImageSummary{}, repo: "busybox"},
			expValue: "busybox",
			call:     ctx.Repository,
		},
		{
			imageCtx: imageContext{i: types.ImageSummary{}, tag: "latest"},
			expValue: "latest",
			call:     ctx.Tag,
		},
		{
			imageCtx: imageContext{i: types.ImageSummary{}, digest: "sha256:d149ab53f8718e987c3a3024bb8aa0e2caadf6c0328f1d9d850b2a2a67f2819a"},
			expValue: "sha256:d149ab53f8718e987c3a3024bb8aa0e2caadf6c0328f1d9d850b2a2a67f2819a",
			call:     ctx.Digest,
		},
		{
			imageCtx: imageContext{i: types.ImageSummary{Containers: 10}},
			expValue: "10",
			call:     ctx.Containers,
		},
		{
			imageCtx: imageContext{i: types.ImageSummary{VirtualSize: 10000}},
			expValue: "10kB",
			call:     ctx.VirtualSize,
		},
		{
			imageCtx: imageContext{i: types.ImageSummary{SharedSize: 10000}},
			expValue: "10kB",
			call:     ctx.SharedSize,
		},
		{
			imageCtx: imageContext{i: types.ImageSummary{SharedSize: 5000, VirtualSize: 20000}},
			expValue: "15kB",
			call:     ctx.UniqueSize,
		},
		{
			imageCtx: imageContext{i: types.ImageSummary{Created: zeroTime}},
			expValue: "",
			call:     ctx.CreatedSince,
		},
	}

	for _, c := range cases {
		ctx = c.imageCtx
		v := c.call()
		if strings.Contains(v, ",") {
			test.CompareMultipleValues(t, v, c.expValue)
		} else {
			assert.Check(t, is.Equal(c.expValue, v))
		}
	}
}

func TestImageContextWrite(t *testing.T) {
	unixTime := time.Now().AddDate(0, 0, -1).Unix()
	zeroTime := int64(-62135596800)
	expectedTime := time.Unix(unixTime, 0).String()
	expectedZeroTime := time.Unix(zeroTime, 0).String()

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
			`REPOSITORY   TAG       IMAGE ID   CREATED        SIZE
image        tag1      imageID1   24 hours ago   0B
image        tag2      imageID2   N/A            0B
<none>       <none>    imageID3   24 hours ago   0B
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
			`REPOSITORY   DIGEST
image        sha256:cbbf2f9a99b47fc460d422812b6a5adff7dfee951d8fa2e4a98caa0382cfbdbf
image        <none>
<none>       <none>
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
			`REPOSITORY   TAG       DIGEST                                                                    IMAGE ID   CREATED        SIZE
image        tag1      sha256:cbbf2f9a99b47fc460d422812b6a5adff7dfee951d8fa2e4a98caa0382cfbdbf   imageID1   24 hours ago   0B
image        tag2      <none>                                                                    imageID2   N/A            0B
<none>       <none>    <none>                                                                    imageID3   24 hours ago   0B
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

`, expectedTime, expectedZeroTime, expectedTime),
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

`, expectedTime, expectedZeroTime, expectedTime),
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

	images := []types.ImageSummary{
		{ID: "imageID1", RepoTags: []string{"image:tag1"}, RepoDigests: []string{"image@sha256:cbbf2f9a99b47fc460d422812b6a5adff7dfee951d8fa2e4a98caa0382cfbdbf"}, Created: unixTime},
		{ID: "imageID2", RepoTags: []string{"image:tag2"}, Created: zeroTime},
		{ID: "imageID3", RepoTags: []string{"<none>:<none>"}, RepoDigests: []string{"<none>@<none>"}, Created: unixTime},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(string(tc.context.Format), func(t *testing.T) {
			var out bytes.Buffer
			tc.context.Output = &out
			err := ImageWrite(tc.context, images)
			if err != nil {
				assert.Error(t, err, tc.expected)
			} else {
				assert.Equal(t, out.String(), tc.expected)
			}
		})
	}
}

func TestImageContextWriteWithNoImage(t *testing.T) {
	out := bytes.NewBufferString("")
	images := []types.ImageSummary{}

	cases := []struct {
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
			"REPOSITORY   DIGEST\n",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(string(tc.context.Format), func(t *testing.T) {
			err := ImageWrite(tc.context, images)
			assert.NilError(t, err)
			assert.Equal(t, out.String(), tc.expected)
			// Clean buffer
			out.Reset()
		})
	}
}
