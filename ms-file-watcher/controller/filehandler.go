/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package controller

import (
	"aicsd/pkg/wait"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	data_organizer "aicsd/ms-file-watcher/clients/data_organizer"
	"aicsd/ms-file-watcher/config"

	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/logger"
	"github.com/fsnotify/fsnotify"
)

type FileHandler struct {
	lc                logger.LoggingClient
	dataOrgClient     data_organizer.Client
	DependentServices wait.Services
	Config            *config.Configuration
}

func New(lc logger.LoggingClient, dataOrgClient data_organizer.Client, Config *config.Configuration) *FileHandler {
	return &FileHandler{
		lc:                lc,
		dataOrgClient:     dataOrgClient,
		DependentServices: wait.Services{wait.ServiceConsul},
		Config:            Config,
	}
}

// WatchFolders is a goroutine that is called to watch a set of folders for files to be created. When a file is created,
// it is checked and if it is included, then the data organizer is notified.
func (fh *FileHandler) WatchFolders(ctx context.Context, wg *sync.WaitGroup, config *config.Configuration) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fh.lc.Errorf("New watcher could not be created: %s", err.Error())
		return
	}
	defer watcher.Close()

	for _, dir := range config.FoldersToWatch {
		err = watcher.Add(dir)
		if err != nil {
			fh.lc.Errorf("WatchFolders could not add folder to watcher: %s", dir)
			return
		}
		fh.walkDirectory(dir, config.App.UpdatableSettings.WatchSubfolders, config.FileExclusionList)
		fh.lc.Debugf("Folder %s walked", dir)
	}

	for {
		select {
		case <-ctx.Done():
			watcher.Close()
			wg.Done()
			fh.lc.Info("Watch Folders Exiting")
			return
		case event := <-watcher.Events:
			if event.Op&fsnotify.Create == fsnotify.Create {
				// TODO: resolve the fact that the file may not be written or closed when it is created!
				fileInfo, err := os.Stat(event.Name)
				if err != nil {
					fh.lc.Errorf("Error checking new file/folder info %s: %s", event.Name, err.Error())
					continue
				}
				if !fileInfo.IsDir() {
					excluded := false
					if len(config.FileExclusionList) != 0 {
						_, filename := filepath.Split(event.Name)

						for _, file := range config.FileExclusionList {
							if strings.Contains(filename, file) {
								excluded = true
								break
							}
						}
					}
					if !excluded {
						fh.lc.Debugf("Create: %s: %s", event.Op, event.Name)
						// todo add job to the notify new file
						err := fh.dataOrgClient.NotifyNewFile(event.Name)
						if err != nil {
							fh.lc.Errorf("Error sending new file notification for file %s: %s", event.Name, err.Error())
							continue
						} else {
							fh.lc.Debugf("Sent new file notification for %s", event.Name)
						}
					} else {
						fh.lc.Debugf("Ignoring specified file: %s", event.Name)
					}
				} else if config.App.UpdatableSettings.WatchSubfolders {
					fh.lc.Debugf("Found new folder at %s", event.Name)
					err := watcher.Add(event.Name)
					if err != nil {
						fh.lc.Errorf("WatchFolders could not add folder to watcher: %s", event.Name)
						return
					}
					fh.lc.Debugf("Walking path at: %s", event.Name)
					fh.walkDirectory(event.Name, config.App.UpdatableSettings.WatchSubfolders, config.FileExclusionList)
					fh.lc.Debugf("Folder %s walked", event.Name)
				}
			}
		case err := <-watcher.Errors:
			fh.lc.Errorf("Error:", err)
		}
	}
}

// walkDirectory is a helper function that is called on startup to help iterate over all the files in a given folder.
// Each file is pre-checked through a file-exclusion filter, the included files are sent to the data organizer,
// so that it may decide if the file needs to be processed or not.
func (fh *FileHandler) walkDirectory(folder string, watchSubfolders bool, fileExclusionList []string) {
	files, err := os.ReadDir(folder)
	if err != nil {
		return
	}
	for _, file := range files {
		filename := filepath.Join(folder, file.Name())
		if !file.IsDir() {
			excluded := false
			if len(fileExclusionList) != 0 {
				for _, entry := range fileExclusionList {
					if strings.Contains(file.Name(), entry) {
						excluded = true
						break
					}
				}
			}
			if !excluded {
				fh.lc.Debugf("Found file %s while walking directory", filename)
				fh.lc.Debugf("Create: CREATE: %s", filename)
				err := fh.dataOrgClient.NotifyNewFile(filename)
				if err != nil {
					fh.lc.Errorf("Error walking directory and sending new file notification for file %s: %s", filename, err.Error())
					// don't return after erroring out here in case it is just an issue with a particular file
				} else {
					fh.lc.Debugf("Sent new file notification for %s", filename)
				}
			} else {
				fh.lc.Debugf("Ignoring specified file: %s", filename)
			}
		} else if watchSubfolders {
			fh.lc.Debugf("Found new folder at %s", file.Name())
			watcher, err := fsnotify.NewWatcher()
			if err != nil {
				fh.lc.Errorf("New watcher could not be created: %s", err.Error())
				return
			}
			defer watcher.Close()

			fh.lc.Debugf("Walking path at: %s", filename)
			err = watcher.Add(filename)
			if err != nil {
				fh.lc.Errorf("WalkDirectory could not add folder to watcher: %s", filename)
				return
			}
			fh.walkDirectory(filename, watchSubfolders, fileExclusionList)
			fh.lc.Debugf("Folder %s walked", filename)
		}
	}
}

func (fh *FileHandler) ProcessConfigUpdates(rawWritableConfig interface{}) {

	updated, ok := rawWritableConfig.(*config.UpdatableSettings)
	if !ok {
		fh.lc.Error("Process config updates failed")
		return
	}

	previous := fh.Config.App
	fh.Config.App.UpdatableSettings = *updated

	if reflect.DeepEqual(previous, updated) {
		fh.lc.Info("No changes detected")
		return
	}

	if previous.UpdatableSettings.WatchSubfolders != updated.WatchSubfolders {
		// TODO: React to this value changing
		fh.lc.Infof("Watch Subfolders set to: %v ", updated.WatchSubfolders)
	}

	if !reflect.DeepEqual(previous.UpdatableSettings.FileExclusionList, updated.FileExclusionList) {
		formatedExclusionList := strings.ReplaceAll(updated.FileExclusionList, " ", "")
		if formatedExclusionList == "" {
			var emptySlice []string
			fh.Config.FileExclusionList = emptySlice
		} else {
			fh.Config.FileExclusionList = strings.Split(formatedExclusionList, ",")
		}
		fh.lc.Infof("File Exclusion List set to: %v", updated.FileExclusionList)
		return
	}
}
