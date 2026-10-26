from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.preview_transcription_batch_response_200 import PreviewTranscriptionBatchResponse200
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    *,
    include_corrected: bool | Unset = UNSET,
    visibility: str | Unset = UNSET,
    status: str | Unset = UNSET,
    owner_id: str | Unset = UNSET,
    source_id: str | Unset = UNSET,
    from_: str | Unset = UNSET,
    to: str | Unset = UNSET,

) -> dict[str, Any]:




    params: dict[str, Any] = {}

    params["include_corrected"] = include_corrected

    params["visibility"] = visibility

    params["status"] = status

    params["owner_id"] = owner_id

    params["source_id"] = source_id

    params["from"] = from_

    params["to"] = to


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/admin/transcriptions/batch-preview",
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | PreviewTranscriptionBatchResponse200:
    if response.status_code == 200:
        response_200 = PreviewTranscriptionBatchResponse200.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | PreviewTranscriptionBatchResponse200]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    include_corrected: bool | Unset = UNSET,
    visibility: str | Unset = UNSET,
    status: str | Unset = UNSET,
    owner_id: str | Unset = UNSET,
    source_id: str | Unset = UNSET,
    from_: str | Unset = UNSET,
    to: str | Unset = UNSET,

) -> Response[Error | PreviewTranscriptionBatchResponse200]:
    """
    Args:
        include_corrected (bool | Unset):
        visibility (str | Unset):
        status (str | Unset):
        owner_id (str | Unset):
        source_id (str | Unset):
        from_ (str | Unset):
        to (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | PreviewTranscriptionBatchResponse200]
     """


    kwargs = _get_kwargs(
        include_corrected=include_corrected,
visibility=visibility,
status=status,
owner_id=owner_id,
source_id=source_id,
from_=from_,
to=to,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient | Client,
    include_corrected: bool | Unset = UNSET,
    visibility: str | Unset = UNSET,
    status: str | Unset = UNSET,
    owner_id: str | Unset = UNSET,
    source_id: str | Unset = UNSET,
    from_: str | Unset = UNSET,
    to: str | Unset = UNSET,

) -> Error | PreviewTranscriptionBatchResponse200 | None:
    """
    Args:
        include_corrected (bool | Unset):
        visibility (str | Unset):
        status (str | Unset):
        owner_id (str | Unset):
        source_id (str | Unset):
        from_ (str | Unset):
        to (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | PreviewTranscriptionBatchResponse200
     """


    return sync_detailed(
        client=client,
include_corrected=include_corrected,
visibility=visibility,
status=status,
owner_id=owner_id,
source_id=source_id,
from_=from_,
to=to,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    include_corrected: bool | Unset = UNSET,
    visibility: str | Unset = UNSET,
    status: str | Unset = UNSET,
    owner_id: str | Unset = UNSET,
    source_id: str | Unset = UNSET,
    from_: str | Unset = UNSET,
    to: str | Unset = UNSET,

) -> Response[Error | PreviewTranscriptionBatchResponse200]:
    """
    Args:
        include_corrected (bool | Unset):
        visibility (str | Unset):
        status (str | Unset):
        owner_id (str | Unset):
        source_id (str | Unset):
        from_ (str | Unset):
        to (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | PreviewTranscriptionBatchResponse200]
     """


    kwargs = _get_kwargs(
        include_corrected=include_corrected,
visibility=visibility,
status=status,
owner_id=owner_id,
source_id=source_id,
from_=from_,
to=to,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    include_corrected: bool | Unset = UNSET,
    visibility: str | Unset = UNSET,
    status: str | Unset = UNSET,
    owner_id: str | Unset = UNSET,
    source_id: str | Unset = UNSET,
    from_: str | Unset = UNSET,
    to: str | Unset = UNSET,

) -> Error | PreviewTranscriptionBatchResponse200 | None:
    """
    Args:
        include_corrected (bool | Unset):
        visibility (str | Unset):
        status (str | Unset):
        owner_id (str | Unset):
        source_id (str | Unset):
        from_ (str | Unset):
        to (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | PreviewTranscriptionBatchResponse200
     """


    return (await asyncio_detailed(
        client=client,
include_corrected=include_corrected,
visibility=visibility,
status=status,
owner_id=owner_id,
source_id=source_id,
from_=from_,
to=to,

    )).parsed
