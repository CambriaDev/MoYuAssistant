//go:build module_qr_transfer

package qrtransfer

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ncruces/zenity"
	"github.com/skip2/go-qrcode"

	"moyu-assistant/internal/i18n"
	"moyu-assistant/internal/module"
)

const defaultFrameInterval = time.Second / 10

func init() {
	module.Register(&QRTransferModule{})
}

type QRTransferModule struct {
	state *appState
}

func (m *QRTransferModule) Name() string {
	return i18n.T("二维码文件传输", "QR File Transfer")
}

func (m *QRTransferModule) Description() string {
	return i18n.T("将本地文件拆分为二维码序列", "Split a local file into a QR code sequence")
}

func (m *QRTransferModule) Icon() fyne.Resource {
	return theme.DocumentIcon()
}

func (m *QRTransferModule) Category() string {
	return i18n.T("文件传输", "File Transfer")
}

func (m *QRTransferModule) OnInit() {
	m.state = &appState{}
}

func (m *QRTransferModule) OnDestroy() {
	m.stopPlayback()
}

type appState struct {
	mu             sync.Mutex
	transfer       *Transfer
	current        int
	frameInterval  time.Duration
	cancelPlayback context.CancelFunc
	openButton     *widget.Button
	playButton     *widget.Button
	jumpButton     *widget.Button
	status         *widget.Label
	frameLabel     *widget.Label
	progress       *widget.ProgressBar
	positionSlider *widget.Slider
	frameInput     *widget.Entry
	qrImage        *canvas.Image
	placeholder    fyne.CanvasObject
}

func (m *QRTransferModule) CreateUI(window fyne.Window) fyne.CanvasObject {
	if m.state == nil {
		m.state = &appState{}
	}

	header := module.CreateHeader(
		i18n.T("二维码文件传输", "QR File Transfer"),
		i18n.T("在桌面显示二维码序列，并在移动端逐帧扫描以还原文件", "Display a QR sequence for a mobile device to scan and reconstruct"),
	)

	status := widget.NewLabel(i18n.T("请选择要传输的文件", "Select a file to transfer"))
	frameLabel := widget.NewLabel(i18n.T("尚未生成二维码", "No QR codes generated"))
	progress := widget.NewProgressBar()
	progress.Hide()
	qrImage := canvas.NewImageFromResource(theme.DocumentIcon())
	qrImage.FillMode = canvas.ImageFillContain
	qrImage.Hide()
	placeholder := container.NewCenter(widget.NewLabel(i18n.T("二维码将在此处显示", "The QR code will appear here")))

	var openButton *widget.Button
	openButton = widget.NewButton(i18n.T("选择文件", "Select File"), func() {
		m.stopPlayback()
		status.SetText(i18n.T("请选择本地文件…", "Select a local file…"))
		openButton.Disable()
		go func() {
			fileName, err := zenity.SelectFile(
				zenity.Title(i18n.T("选择要传输的文件", "Select a file to transfer")),
			)
			if err != nil {
				if err == zenity.ErrCanceled {
					m.setStatus(i18n.T("未选择文件", "No file selected"))
				} else {
					m.setStatus(i18n.T("选择文件失败", "Failed to select file") + ": " + err.Error())
				}
				m.enableFilePicker()
				return
			}
			if fileName == "" {
				m.setStatus(i18n.T("未选择文件", "No file selected"))
				m.enableFilePicker()
				return
			}
			m.setStatus(i18n.T("正在读取文件…", "Reading file…"))
			go m.loadFile(fileName)
		}()
	})

	previousButton := widget.NewButton(i18n.T("上一帧", "Previous"), func() {
		m.moveFrame(-1)
	})
	playButton := widget.NewButton(i18n.T("自动播放", "Play"), m.togglePlayback)
	nextButton := widget.NewButton(i18n.T("下一帧", "Next"), func() {
		m.moveFrame(1)
	})
	frameInput := widget.NewEntry()
	frameInput.SetText("1")
	frameInput.SetPlaceHolder(i18n.T("二维码序号", "QR number"))
	frameInput.Hide()
	jumpToFrame := func(value string) {
		position, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			m.setStatus(i18n.T("请输入有效的二维码序号", "Enter a valid QR number"))
			m.restoreFrameInput()
			return
		}
		m.seekToPosition(position)
	}
	frameInput.OnSubmitted = jumpToFrame
	jumpButton := widget.NewButton(i18n.T("跳转", "Go"), func() {
		jumpToFrame(frameInput.Text)
	})
	jumpButton.Disable()
	positionSlider := widget.NewSlider(1, 2)
	positionSlider.Step = 1
	positionSlider.SetValue(1)
	positionSlider.Hide()
	positionSlider.OnChanged = func(value float64) {
		m.seekToPosition(int(value))
	}
	frameRateOptions := []string{
		i18n.T("10 FPS（默认）", "10 FPS (default)"),
		i18n.T("2 FPS", "2 FPS"),
		i18n.T("1 FPS", "1 FPS"),
		i18n.T("0.5 FPS", "0.5 FPS"),
		i18n.T("0.2 FPS", "0.2 FPS"),
	}
	frameIntervals := map[string]time.Duration{
		frameRateOptions[0]: time.Second / 10,
		frameRateOptions[1]: time.Second / 2,
		frameRateOptions[2]: time.Second,
		frameRateOptions[3]: 2 * time.Second,
		frameRateOptions[4]: 5 * time.Second,
	}
	frameRateSelect := widget.NewSelect(frameRateOptions, nil)
	frameRateSelect.SetSelected(frameRateOptions[0])
	frameRateSelect.OnChanged = func(selected string) {
		m.state.mu.Lock()
		wasPlaying := m.state.cancelPlayback != nil
		m.state.mu.Unlock()
		if wasPlaying {
			m.stopPlayback()
		}
		interval := frameIntervals[selected]
		m.state.mu.Lock()
		m.state.frameInterval = interval
		current := m.state.current
		m.state.mu.Unlock()
		m.showFrame(current)
		if wasPlaying {
			m.togglePlayback()
		}
	}

	m.state.mu.Lock()
	m.state.frameInterval = defaultFrameInterval
	m.state.openButton = openButton
	m.state.playButton = playButton
	m.state.jumpButton = jumpButton
	m.state.status = status
	m.state.frameLabel = frameLabel
	m.state.progress = progress
	m.state.positionSlider = positionSlider
	m.state.frameInput = frameInput
	m.state.qrImage = qrImage
	m.state.placeholder = placeholder
	m.state.mu.Unlock()

	preview := container.NewMax(placeholder, container.NewGridWrap(fyne.NewSize(420, 420), qrImage))
	controls := container.NewHBox(previousButton, playButton, nextButton, frameRateSelect)
	positionControls := container.NewBorder(
		nil,
		nil,
		widget.NewLabel(i18n.T("二维码序号", "QR number")),
		container.NewHBox(frameInput, jumpButton),
		positionSlider,
	)
	footer := container.NewVBox(frameLabel, progress, positionControls, container.NewCenter(controls))
	top := container.NewVBox(header, container.NewCenter(openButton))

	return container.NewBorder(
		top,
		container.NewVBox(status, footer),
		nil,
		nil,
		container.NewCenter(preview),
	)
}

func (m *QRTransferModule) loadFile(fileName string) {
	file, err := os.Open(fileName)
	if err != nil {
		m.setStatus(i18n.T("打开文件失败", "Failed to open file") + ": " + err.Error())
		m.enableFilePicker()
		return
	}
	data, readErr := io.ReadAll(file)
	closeErr := file.Close()
	if readErr != nil {
		m.setStatus(i18n.T("读取文件失败", "Failed to read file") + ": " + readErr.Error())
		m.enableFilePicker()
		return
	}
	if closeErr != nil {
		m.setStatus(i18n.T("关闭文件失败", "Failed to close file") + ": " + closeErr.Error())
		m.enableFilePicker()
		return
	}

	transfer, err := NewTransfer(fileName, data)
	if err != nil {
		m.setStatus(i18n.T("准备传输失败", "Could not prepare transfer") + ": " + err.Error())
		m.enableFilePicker()
		return
	}

	m.state.mu.Lock()
	m.state.transfer = transfer
	m.state.current = -1
	positionSlider := m.state.positionSlider
	frameInput := m.state.frameInput
	jumpButton := m.state.jumpButton
	m.state.mu.Unlock()

	positionSlider.Max = float64(transfer.Total + 1)
	positionSlider.SetValue(1)
	positionSlider.Show()
	frameInput.SetText("1")
	frameInput.SetPlaceHolder(fmt.Sprintf("1-%d", transfer.Total+1))
	frameInput.Show()
	jumpButton.Enable()
	m.enableFilePicker()
	m.showFrame(-1)
	m.setStatus(fmt.Sprintf(
		i18n.T("%s · %d 字节 · %d 个数据二维码，另有 metadata", "%s · %d bytes · %d data QR codes plus metadata"),
		transfer.Name,
		transfer.Size,
		transfer.Total,
	))
}

func (m *QRTransferModule) showFrame(index int) {
	m.state.mu.Lock()
	transfer := m.state.transfer
	qrImage := m.state.qrImage
	placeholder := m.state.placeholder
	frameLabel := m.state.frameLabel
	progress := m.state.progress
	positionSlider := m.state.positionSlider
	frameInput := m.state.frameInput
	interval := m.state.frameInterval
	playing := m.state.cancelPlayback != nil
	m.state.mu.Unlock()
	if transfer == nil || qrImage == nil {
		return
	}

	var (
		payload string
		err     error
	)
	if index < 0 {
		payload, err = transfer.ManifestPayload()
	} else {
		payload, err = transfer.Payload(index)
	}
	if err != nil {
		m.setStatus(i18n.T("生成二维码失败", "Failed to generate QR code") + ": " + err.Error())
		return
	}
	pngData, err := qrcode.Encode(payload, qrcode.Medium, 480)
	if err != nil {
		m.setStatus(i18n.T("生成二维码失败", "Failed to generate QR code") + ": " + err.Error())
		return
	}

	qrImage.Resource = fyne.NewStaticResource("qr-transfer.png", pngData)
	qrImage.Show()
	placeholder.Hide()
	position := index + 2
	positionSlider.SetValue(float64(position))
	if !playing {
		frameInput.SetText(strconv.Itoa(position))
	}
	if index < 0 {
		frameLabel.SetText(fmt.Sprintf(
			i18n.T("二维码 %d / %d · Metadata · 播放速度 %s", "QR %d / %d · metadata · speed %s"),
			position,
			transfer.Total+1,
			frameRateLabel(interval),
		))
		progress.SetValue(0)
	} else {
		frameLabel.SetText(fmt.Sprintf(
			i18n.T("二维码 %d / %d · 数据分片 %d / %d · %s", "QR %d / %d · data chunk %d / %d · %s"),
			position,
			transfer.Total+1,
			index+1,
			transfer.Total,
			frameRateLabel(interval),
		))
		progress.SetValue(float64(index+1) / float64(transfer.Total))
	}
	progress.Show()
	qrImage.Refresh()
}

func (m *QRTransferModule) moveFrame(delta int) {
	m.stopPlayback()
	m.state.mu.Lock()
	if m.state.transfer == nil {
		m.state.mu.Unlock()
		return
	}
	index := m.state.current + delta
	if index < -1 {
		index = -1
	}
	if index >= m.state.transfer.Total {
		index = m.state.transfer.Total - 1
	}
	m.state.current = index
	m.state.mu.Unlock()
	m.showFrame(index)
}

func (m *QRTransferModule) seekToPosition(position int) {
	m.state.mu.Lock()
	transfer := m.state.transfer
	current := m.state.current
	frameInput := m.state.frameInput
	m.state.mu.Unlock()
	if transfer == nil {
		return
	}

	maxPosition := transfer.Total + 1
	if position < 1 || position > maxPosition {
		frameInput.SetText(strconv.Itoa(current + 2))
		m.setStatus(fmt.Sprintf(
			i18n.T("请输入 1 到 %d 之间的二维码序号", "Enter a QR number between 1 and %d"),
			maxPosition,
		))
		return
	}

	index := position - 2
	if index == current {
		return
	}
	m.stopPlayback()
	m.state.mu.Lock()
	m.state.current = index
	m.state.mu.Unlock()
	m.showFrame(index)
}

func (m *QRTransferModule) restoreFrameInput() {
	m.state.mu.Lock()
	position := m.state.current + 2
	frameInput := m.state.frameInput
	m.state.mu.Unlock()
	if frameInput != nil {
		frameInput.SetText(strconv.Itoa(position))
	}
}

func frameRateLabel(interval time.Duration) string {
	switch interval {
	case time.Second / 10:
		return "10 FPS"
	case time.Second / 2:
		return "2 FPS"
	case time.Second:
		return "1 FPS"
	case 2 * time.Second:
		return "0.5 FPS"
	case 5 * time.Second:
		return "0.2 FPS"
	default:
		return ""
	}
}

func (m *QRTransferModule) togglePlayback() {
	m.state.mu.Lock()
	if m.state.cancelPlayback != nil {
		cancel := m.state.cancelPlayback
		m.state.cancelPlayback = nil
		playButton := m.state.playButton
		m.state.mu.Unlock()
		cancel()
		playButton.SetText(i18n.T("自动播放", "Play"))
		m.restoreFrameInput()
		return
	}
	if m.state.transfer == nil {
		m.state.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.state.cancelPlayback = cancel
	playButton := m.state.playButton
	m.state.mu.Unlock()

	playButton.SetText(i18n.T("暂停播放", "Pause"))
	go func() {
		for {
			m.state.mu.Lock()
			interval := m.state.frameInterval
			m.state.mu.Unlock()
			timer := time.NewTimer(interval)
			select {
			case <-ctx.Done():
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				return
			case <-timer.C:
				m.state.mu.Lock()
				if m.state.transfer == nil {
					m.state.cancelPlayback = nil
					m.state.mu.Unlock()
					playButton.SetText(i18n.T("自动播放", "Play"))
					return
				}
				m.state.current = (m.state.current+2)%(m.state.transfer.Total+1) - 1
				index := m.state.current
				m.state.mu.Unlock()
				m.showFrame(index)
			}
		}
	}()
}

func (m *QRTransferModule) stopPlayback() {
	m.state.mu.Lock()
	cancel := m.state.cancelPlayback
	playButton := m.state.playButton
	m.state.cancelPlayback = nil
	m.state.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if playButton != nil {
		playButton.SetText(i18n.T("自动播放", "Play"))
	}
	m.restoreFrameInput()
}

func (m *QRTransferModule) enableFilePicker() {
	m.state.mu.Lock()
	openButton := m.state.openButton
	m.state.mu.Unlock()
	if openButton != nil {
		openButton.Enable()
	}
}

func (m *QRTransferModule) setStatus(message string) {
	m.state.mu.Lock()
	status := m.state.status
	m.state.mu.Unlock()
	if status != nil {
		status.SetText(message)
	}
}
