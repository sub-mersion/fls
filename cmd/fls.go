package cmd

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	scale      float32
	outputPath string
	verbose    bool
)

var rootCmd = &cobra.Command{
	Use:   "fls <input_file>",
	Short: "fls produces paletted black and white images using the Floyd-Steinberg dithering algorithm.",
	Long: `fls produces paletted black and white images using the Floyd-Steinberg dithering
algorithm. It is a simple wrapper around the built-in function of the
golang.org/x/image/draw package. Rescaling is applied before the dithering with
the nearest-neighbor algorithm.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		path := filepath.Clean(args[0])
		log.Info().Msgf("read file %q", path)
		data, err := ioutil.ReadFile(path)
		if err != nil {
			log.Fatal().Err(err).Msgf("reading file %q", path)
		}

		var img image.Image

		switch filepath.Ext(path) {
		case ".png":
			img, err = png.Decode(bytes.NewBuffer(data))
			if err != nil {
				log.Fatal().Err(err).Msgf("decoding png image %q", path)
			}
		case ".jpg", ".jpeg":
			img, err = jpeg.Decode(bytes.NewBuffer(data))
			if err != nil {
				log.Fatal().Err(err).Msgf("decoding jpeg image %q", path)
			}
		default:
			log.Fatal().Err(err).Msgf("image type %s not supported", filepath.Ext(path))
		}

		palette := color.Palette{color.White, color.Black}
		rect := img.Bounds()
		if scale != 1. {
			log.Info().Float32("scale", scale).Msg("resizing")
			rect = image.Rect(0, 0, int(float32(rect.Dx())*scale), int(float32(rect.Dy())*scale))
			tmp := image.NewRGBA(rect)
			draw.NearestNeighbor.Scale(tmp, rect, img, img.Bounds(), draw.Over, nil)
			img = tmp
		}
		dst := image.NewPaletted(rect, palette)

		log.Info().Msg("applying Floyd-Steinberg dithering...")
		draw.FloydSteinberg.Draw(dst, img.Bounds(), img, image.Point{})

		if outputPath == "" {
			outputPath = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)) + "_fls.png"
		}
		log.Info().Msgf("writing result PNG image at path %q", outputPath)
		file, err := os.Create(outputPath)
		if err != nil {
			log.Fatal().Err(err).Msgf("creating output file %q", outputPath)
		}
		defer file.Close()
		if err := png.Encode(file, dst); err != nil {
			log.Fatal().Err(err).Msgf("writing png image in %q", outputPath)
		}
	},
}

func init() {
	rootCmd.PersistentFlags().Float32VarP(&scale, "scale", "s", 1., "Scaling coefficient")
	rootCmd.PersistentFlags().StringVarP(&outputPath, "output", "o", "", "Path to output file")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Set verbose execution")

	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
		if verbose {
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		}
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}
