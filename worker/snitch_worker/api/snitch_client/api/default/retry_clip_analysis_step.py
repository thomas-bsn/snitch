from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.clip_analysis_response import ClipAnalysisResponse
from ...models.error import Error
from ...models.retry_clip_analysis_step_step import RetryClipAnalysisStepStep
from typing import cast



def _get_kwargs(
    id: int,
    step: RetryClipAnalysisStepStep,
    *,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/admin/clips/{id}/analysis/steps/{step}/retry".format(id=quote(str(id), safe=""),step=quote(str(step), safe=""),),
    }


    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> ClipAnalysisResponse | Error:
    if response.status_code == 200:
        response_200 = ClipAnalysisResponse.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[ClipAnalysisResponse | Error]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    id: int,
    step: RetryClipAnalysisStepStep,
    *,
    client: AuthenticatedClient | Client,
    origin: str,
    x_csrf_token: str,

) -> Response[ClipAnalysisResponse | Error]:
    """  Administrateurs. Relance une étape en échec définitif : son job échoué est ignoré et remplacé, les
    résultats valides sont conservés et la politique de retry des jobs s'applique à nouveau.

    Args:
        id (int):
        step (RetryClipAnalysisStepStep):
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ClipAnalysisResponse | Error]
     """


    kwargs = _get_kwargs(
        id=id,
step=step,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    id: int,
    step: RetryClipAnalysisStepStep,
    *,
    client: AuthenticatedClient | Client,
    origin: str,
    x_csrf_token: str,

) -> ClipAnalysisResponse | Error | None:
    """  Administrateurs. Relance une étape en échec définitif : son job échoué est ignoré et remplacé, les
    résultats valides sont conservés et la politique de retry des jobs s'applique à nouveau.

    Args:
        id (int):
        step (RetryClipAnalysisStepStep):
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ClipAnalysisResponse | Error
     """


    return sync_detailed(
        id=id,
step=step,
client=client,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    id: int,
    step: RetryClipAnalysisStepStep,
    *,
    client: AuthenticatedClient | Client,
    origin: str,
    x_csrf_token: str,

) -> Response[ClipAnalysisResponse | Error]:
    """  Administrateurs. Relance une étape en échec définitif : son job échoué est ignoré et remplacé, les
    résultats valides sont conservés et la politique de retry des jobs s'applique à nouveau.

    Args:
        id (int):
        step (RetryClipAnalysisStepStep):
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ClipAnalysisResponse | Error]
     """


    kwargs = _get_kwargs(
        id=id,
step=step,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    id: int,
    step: RetryClipAnalysisStepStep,
    *,
    client: AuthenticatedClient | Client,
    origin: str,
    x_csrf_token: str,

) -> ClipAnalysisResponse | Error | None:
    """  Administrateurs. Relance une étape en échec définitif : son job échoué est ignoré et remplacé, les
    résultats valides sont conservés et la politique de retry des jobs s'applique à nouveau.

    Args:
        id (int):
        step (RetryClipAnalysisStepStep):
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ClipAnalysisResponse | Error
     """


    return (await asyncio_detailed(
        id=id,
step=step,
client=client,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
