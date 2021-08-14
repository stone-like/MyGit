package src

import (
	"fmt"
	"io"
	con "mygit/src/database/content"
	"mygit/src/database/lock"
	ers "mygit/src/errors"
)

// func WriteCommitProcess(cp *CherryPick) error {
// 	mergeType, err := cp.pc.GetMergeType()
// 	if err != nil {
// 		return err
// 	}
// 	switch mergeType {
// 	case PEDING_CHERRY_PICK_TYPE:
// 		return WriteCherryPickCommit(cp.pc, cp.repo)
// 	case PENDING_REVERT_TYPE:
// 		return WriteRevertCommit(cp.pc, cp.repo)
// 	default:
// 		return ErrorInvalidMergeType
// 	}
// }

func ContinueCommitProcess(sd *SequenceData, sc SequencingClass) error {
	l := lock.NewFileLock(sd.repo.i.Path)
	l.Lock()
	defer l.Unlock()

	err := sd.repo.i.Load()
	if err != nil {
		return err
	}
	err = sc.ContinueWriteCommit(sd)
	if err != nil {
		return err
	}

	return nil
}

func HandleSequencingContinue(sd *SequenceData, sc SequencingClass) error {

	err := ContinueCommitProcess(sd, sc)
	if err != nil {
		return err
	}

	//toDoPathをLock
	sl := lock.NewFileLock(sd.seq.GetToDoPath())
	sl.Lock()
	defer sl.Unlock()

	err = sd.seq.Load()
	if err != nil {
		return err
	}

	//RunPick中にエラーが出ると、shiftをする前でエラーとしてreturnされるので、まず操作完了させるためにshiftから
	sd.seq.Shift()

	//seq処理を再開
	return ResumeSeq(sd)

}

func RunCommand(command *CommandContent, sd *SequenceData) error {
	switch command.Type {
	case PICK:
		return RunPick(command.c, sd)

	case REVERT:
		return RunRevert(command.c, sd)
	default:
		return ErrorInvalidToDoContent
	}

}

func ResumeSeq(sd *SequenceData) error {

	for {

		nextCommand := sd.seq.NextCommand()

		if nextCommand == nil {
			break
		}

		err := RunCommand(nextCommand, sd)
		if err != nil {
			return err
		}

		//正常にPickできたらShiftでSeqから取り除く
		_, err = sd.seq.Shift()
		if err != nil {
			return err
		}

		//正常にpickできたら、abortSafetyを最新のコミットにupdate
		err = sd.seq.UpdateAbortSafetyLatest()
		if err != nil {
			return err
		}
	}

	//エラーなしでできたら.git/sequencerを消す
	sd.seq.Clear()

	return nil
}

//abortはquitに加えてcherryPickをする前に戻す、なのでsequencer.abortでresetHardと、updateRefを使用
func HandleSequencingAbort(sd *SequenceData, sc SequencingClass) error {
	if sd.pc.InProgress() {
		err := sd.pc.Clear(sc.GetPendingType())
		if err != nil {
			return err
		}
	}

	l := lock.NewFileLock(sd.repo.i.Path)
	l.Lock()
	defer l.Unlock()

	err := sd.repo.i.Load()
	if err != nil {
		return err
	}

	err = sd.seq.Abort()
	if err != nil {
		return err
	}

	//indexもアップデート
	err = sd.repo.i.Write(sd.repo.i.Path)
	if err != nil {
		return err
	}

	return nil
}

// sequencing内にあるPEINGCHAERRYをrevertと両方対応可能なようにする

//quitでやることはcherryPickで作ったファイル群を消すことだけ、ただindex,workspace,HEADはcherryPickの影響が残ったまま
func HandleSequencingQuit(sd *SequenceData, sc SequencingClass) error {
	//quit前にConflict中だったら
	if sd.pc.InProgress() {
		err := sd.pc.Clear(sc.GetPendingType())
		if err != nil {
			return err
		}
	}

	return sd.seq.Clear()

}

//cherryはpendingMessageがc.Message,revertはrevertCommitMessageでつくったやつ
func HandleConflict(toPendsingMessage string, pendingType PendingType, c *con.CommitFromMem, m *Merge, sd *SequenceData) error {

	//seqをつかってToDoへ書き込み
	err := sd.seq.WriteToDo()
	if err != nil {
		return err
	}

	message := CreateConflictMessage(toPendsingMessage, sd.repo)
	err = sd.pc.Start(m.rightObjId, message, pendingType)
	if err != nil {
		return err
	}

	return CreateConflictMessageError(m.rightName, ConflictMessage)
}

var ConflictMessage = `hint: 
 after resolving the conflicts, mark the corrected paths
 with 'mygit add <paths>' or 'mygit rm <paths>'
 and commit the result with 'mygit commit'
`

func CreateConflictMessageError(rightName, message string) error {
	return &ers.MergeFailOnConflictError{
		Message: fmt.Sprintf("error: could not apply %s\n%s", rightName, message),
	}
}

func HandleInProgress(m *Merge) error {
	return CreateConflictMessageError(m.rightName, ConflictMessage)
}

type SequencingClass interface {
	StoreCommitToSeq(sd *SequenceData) error
	ContinueWriteCommit(sd *SequenceData) error
	GetPendingType() PendingType
}

type SequenceOption struct {
	hasContinue bool
	hasAbort    bool
	hasQuit     bool
}

type SequenceData struct {
	args   []string
	repo   *Repository
	seq    *Sequencer
	pc     *PendingCommit
	option *SequenceOption
	w      io.Writer
}

func GenerateSequenceData(args []string, seq *Sequencer, pc *PendingCommit, repo *Repository, option *SequenceOption, w io.Writer) *SequenceData {
	return &SequenceData{
		args:   args,
		repo:   repo,
		seq:    seq,
		pc:     pc,
		option: option,
		w:      w,
	}
}

func FinishRunCommit(c *con.Commit, repo *Repository) error {
	err := WriteCommit(c, repo)
	if err != nil {
		return err
	}
	PrintCommit(c)
	return nil
}

func RunSequencing(sd *SequenceData, sc SequencingClass) error {

	if sd.option.hasQuit {
		return HandleSequencingQuit(sd, sc)
	}

	if sd.option.hasAbort {
		return HandleSequencingAbort(sd, sc)
	}

	if sd.option.hasContinue {
		return HandleSequencingContinue(sd, sc)
	}

	err := sd.seq.Start()
	if err != nil {
		return err
	}

	//toDoPathをLock,toDoはCherrypick失敗時、つまりresumrSeqの中のhandleConflictで書かれるのでここでToDoをロック(ResumeSeqの中の方がいいのかもしれないが)
	l := lock.NewFileLock(sd.seq.GetToDoPath())
	l.Lock()
	defer l.Unlock()

	//実際にcherryPick,revertを始める前にStoreでTODOCOmmitを全部Seq.commandに追加している
	//Pick成功時にはcommandから取り除く
	//なのでエラー時にseq.Commandのやつを書き込めばtodo予定の奴だけ書き込まれる、成功済みは書き込まれない
	err = sc.StoreCommitToSeq(sd)
	if err != nil {
		return err
	}

	return ResumeSeq(sd)

}
