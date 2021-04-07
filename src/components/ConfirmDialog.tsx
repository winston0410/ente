import React from 'react';
import constants from 'utils/strings/constants';
import MessageDialog from './MessageDialog';

export enum CONFIRM_ACTION {
    LOGOUT,
    DELETE,
    SESSION_EXPIRED,
    DOWNLOAD_APP,
    CANCEL_SUBSCRIPTION,
}

const CONFIRM_ACTION_VALUES = [
    { text: 'LOGOUT', type: 'danger' },
    { text: 'DELETE', type: 'danger' },
    { text: 'SESSION_EXPIRED', type: 'primary' },
    { text: 'DOWNLOAD_APP', type: 'success' },
    { text: 'CANCEL_SUBSCRIPTION', type: 'danger' },
];

interface Props {
    callback: any;
    action: CONFIRM_ACTION;
    show: boolean;
    onHide: () => void;
}
function ConfirmDialog({ callback, action, ...props }: Props) {
    return (
        <>
            <MessageDialog
                {...props}
                attributes={{
                    title:
                        constants[
                            `${CONFIRM_ACTION_VALUES[action]?.text}_MESSAGE`
                        ],
                    proceed: {
                        text: constants[CONFIRM_ACTION_VALUES[action]?.text],
                        action: callback,
                        variant: CONFIRM_ACTION_VALUES[action]?.type,
                    },
                    close: action !== CONFIRM_ACTION.SESSION_EXPIRED && {
                        text: constants.CLOSE,
                    },
                    staticBackdrop: action === CONFIRM_ACTION.SESSION_EXPIRED,
                }}
            />
        </>
    );
}
export default ConfirmDialog;
