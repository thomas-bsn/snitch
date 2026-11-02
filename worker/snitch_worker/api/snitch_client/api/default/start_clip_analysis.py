from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.clip_analysis_conflict import ClipAnalysisConflict
from ...models.clip_analysis_response import ClipAnalysisResponse
from ...models.error import Error
from ...models.start_clip_analysis_body import StartClipAnalysisBody
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    id: int,
    *,
    body: StartClipAnalysisBody | Unset = UNSET,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/admin/clips/{id}/analysis".format(id=quote(str(id), safe=""),),
    }


    if not isinstance(body, Unset):
        _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> ClipAnalysisConflict | ClipAnalysisResponse | Error:
    if response.status_code == 200:
        response_200 = ClipAnalysisResponse.from_dict(response.json())



        return response_200

    if response.status_code == 201:
        response_201 = ClipAnalysisResponse.from_dict(response.json())



        return response_201

    if response.status_code == 409:
        response_409 = ClipAnalysisConflict.from_dict(response.json())



        return response_409

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[ClipAnalysisConflict | ClipAnalysisResponse | Error]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    id: int,
    *,
    client: AuthenticatedClient | Client,
    body: StartClipAnalysisBody | Unset = UNSET,
    origin: str,
    x_csrf_token: str,

) -> Response[ClipAnalysisConflict | ClipAnalysisResponse | Error]:
    """  Administrateurs. Lance une analyse durable du clip (préparation puis transcription) ; une seule
    analyse active par clip, un second appel renvoie l'analyse existante. Les résultats valides sont
    réutilisés.

    Args:
        id (int):
        origin (str):
        x_csrf_token (str):
        body (StartClipAnalysisBody | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ClipAnalysisConflict | ClipAnalysisResponse | Error]
     """


    kwargs = _get_kwargs(
        id=id,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    id: int,
    *,
    client: AuthenticatedClient | Client,
    body: StartClipAnalysisBody | Unset = UNSET,
    origin: str,
    x_csrf_token: str,

) -> ClipAnalysisConflict | ClipAnalysisResponse | Error | None:
    """  Administrateurs. Lance une analyse durable du clip (préparation puis transcription) ; une seule
    analyse active par clip, un second appel renvoie l'analyse existante. Les résultats valides sont
    réutilisés.

    Args:
        id (int):
        origin (str):
        x_csrf_token (str):
        body (StartClipAnalysisBody | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ClipAnalysisConflict | ClipAnalysisResponse | Error
     """


    return sync_detailed(
        id=id,
client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    id: int,
    *,
    client: AuthenticatedClient | Client,
    body: StartClipAnalysisBody | Unset = UNSET,
    origin: str,
    x_csrf_token: str,

) -> Response[ClipAnalysisConflict | ClipAnalysisResponse | Error]:
    """  Administrateurs. Lance une analyse durable du clip (préparation puis transcription) ; une seule
    analyse active par clip, un second appel renvoie l'analyse existante. Les résultats valides sont
    réutilisés.

    Args:
        id (int):
        origin (str):
        x_csrf_token (str):
        body (StartClipAnalysisBody | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ClipAnalysisConflict | ClipAnalysisResponse | Error]
     """


    kwargs = _get_kwargs(
        id=id,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    id: int,
    *,
    client: AuthenticatedClient | Client,
    body: StartClipAnalysisBody | Unset = UNSET,
    origin: str,
    x_csrf_token: str,

) -> ClipAnalysisConflict | ClipAnalysisResponse | Error | None:
    """  Administrateurs. Lance une analyse durable du clip (préparation puis transcription) ; une seule
    analyse active par clip, un second appel renvoie l'analyse existante. Les résultats valides sont
    réutilisés.

    Args:
        id (int):
        origin (str):
        x_csrf_token (str):
        body (StartClipAnalysisBody | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ClipAnalysisConflict | ClipAnalysisResponse | Error
     """


    return (await asyncio_detailed(
        id=id,
client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
